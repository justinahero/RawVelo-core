package socks

import (
	"context"
	"rawvelo/internal/client"
)

type Handler struct {
	client *client.Client
	ctx    context.Context
}
