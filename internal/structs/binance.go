package structs

import "encoding/json"

type TradePayload struct {
	EventType                string `json:"e"`
	EventTime                int    `json:"E"`
	Symbol                   string `json:"s"`
	TradeId                  int    `json:"t"`
	Price                    string `json:"p"`
	Quantity                 string `json:"q"`
	BuyerOrderId             int    `json:"b"`
	SellerOrderId            int    `json:"a"`
	TradeTime                int    `json:"T"`
	IsTheBuyerTheMarketMaker bool   `json:"m"`
	Ignore                   bool   `json:"M"`
}

type ResponseStreamPayload struct {
	Stream string       `json:"stream"`
	Data   TradePayload `json:"data"`
}

type SendMessagePayload struct {
	Method string   `json:"method"`
	Params []string `json:"params,omitempty"`
	ID     uint32   `json:"id"`
}

type ReceiveMessagePayload struct {
	Result json.RawMessage `json:"result"`
	ID     int             `json:"id"`
}
