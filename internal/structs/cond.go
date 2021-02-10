package structs

import (
	"sync"
)

type ResponseCond struct {
	Lock *sync.Mutex
	Cond *sync.Cond
}
