package binance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	. "github.com/masihyeganeh/binance-watch/internal/structs"
	"github.com/pkg/errors"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Binance struct {
	conn         *websocket.Conn
	symbols      map[string]bool
	sendQueue    chan SendMessagePayload
	sendQueueId  uint32
	receiveQueue chan ResponseStreamPayload
	conds        map[uint32]ResponseCond
	responses    map[uint32]json.RawMessage
}

func New(symbols map[string]bool) (*Binance, error) {
	streams := getStreamsFromSymbols(symbols, "trade")

	u := url.URL{
		Scheme:   "wss",
		Host:     "stream.binance.com:9443",
		Path:     "/stream",
		RawQuery: "streams=" + strings.Join(streams, "/"),
	}
	log.Printf("Connecting to %s websocket", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "error while dialing")
	}

	return &Binance{
		conn:         c,
		symbols:      symbols,
		sendQueue:    make(chan SendMessagePayload, SendQueueSize),
		sendQueueId:  0,
		receiveQueue: make(chan ResponseStreamPayload, ReceiveQueueSize),
		conds:        make(map[uint32]ResponseCond),
		responses:    make(map[uint32]json.RawMessage),
	}, nil
}

func (b *Binance) Start(done chan struct{}, interrupt chan os.Signal) error {
	go b.startReading(done)
	for {
		select {
		case <-done:
			return nil
		case message := <-b.sendQueue:
			err := b.conn.WriteJSON(message)
			if err != nil {
				return errors.Wrap(err, "error while writing")
			}
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := b.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				return errors.Wrap(err, "error while closing")
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return nil
		}
	}
}

func (b *Binance) Receive() chan ResponseStreamPayload {
	return b.receiveQueue
}

func (b *Binance) sendMessage(method string, params []string) json.RawMessage {
	id := atomic.AddUint32(&b.sendQueueId, 1)
	lock := &sync.Mutex{}
	lock.Lock()

	cond := sync.NewCond(lock)
	b.conds[id] = ResponseCond{
		Lock: lock,
		Cond: cond,
	}
	b.sendQueue <- SendMessagePayload{
		Method: method,
		Params: params,
		ID:     id,
	}
	cond.Wait()
	lock.Unlock()

	result := b.responses[id]
	delete(b.conds, id)
	delete(b.responses, id)
	return result
}

// Blocking
func (b *Binance) WatchSymbol(symbol string) error {
	if len(b.symbols) >= 1204 {
		return errors.New("maximum number of symbols reached")
	}
	symbol = strings.ToLower(strings.TrimSpace(symbol))
	resp := b.sendMessage("SUBSCRIBE", []string{symbol + "@trade"})
	if jsonData, err := json.Marshal(resp); err == nil && bytes.Equal(jsonData, []byte("null")) {
		return nil
	}
	return errors.New(fmt.Sprintf("could not watch %q", symbol))
}

// Blocking
func (b *Binance) UnwatchSymbol(symbol string) error {
	symbol = strings.ToLower(strings.TrimSpace(symbol))
	if _, ok := b.symbols[symbol]; !ok {
		return errors.New(fmt.Sprintf("%q is not watched", symbol))
	}
	resp := b.sendMessage("UNSUBSCRIBE", []string{symbol + "@trade"})
	if jsonData, err := json.Marshal(resp); err == nil && bytes.Equal(jsonData, []byte("null")) {
		return nil
	}
	return errors.New(fmt.Sprintf("could not unwatch %q", symbol))

}

func (b *Binance) startReading(done chan struct{}) {
	defer close(done)
	for {
		_, message, err := b.conn.ReadMessage()
		if err != nil {
			log.Println("error while reading:", err)
			return
		}

		var msg ReceiveMessagePayload
		if err = json.Unmarshal(message, &msg); err == nil && msg.ID > 0 {
			msgId := uint32(msg.ID)
			if _, ok := b.conds[msgId]; ok {
				b.conds[msgId].Lock.Lock()
				b.responses[msgId] = msg.Result
				b.conds[msgId].Cond.Broadcast()
				b.conds[msgId].Lock.Unlock()
			}
			continue
		}

		var res ResponseStreamPayload
		if err = json.Unmarshal(message, &res); err != nil {
			log.Printf("received unhandled message: %s", message)
			continue
		}

		b.receiveQueue <- res
	}
}

func (b *Binance) Close() error {
	if b.conn != nil {
		return b.conn.Close()
	}
	return nil
}

func getStreamsFromSymbols(symbols map[string]bool, eventType string) []string {
	result := make([]string, len(symbols))

	i := 0
	for symbol := range symbols {
		result[i] = fmt.Sprintf("%s@%s", symbol, eventType)
		i++
	}

	return result
}
