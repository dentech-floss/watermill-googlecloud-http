package googlecloud_http

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type Subscriber struct {
	config SubscriberConfig

	logger watermill.LoggerAdapter

	outputChannels     []chan *message.Message
	outputChannelsLock sync.Locker

	closed bool
}

// Register the route `pattern` that matches http method to execute the `handler`.
type RegisterHttpHandler func(pattern string, handler http.Handler)

type SubscriberConfig struct {
	RegisterHttpHandler  RegisterHttpHandler
	UnmarshalMessageFunc UnmarshalMessageFunc
}

func NewSubscriber(
	config SubscriberConfig,
	logger watermill.LoggerAdapter,
) (*Subscriber, error) {
	config.setDefaults()

	if config.RegisterHttpHandler == nil {
		return nil, errors.New("missing a 'RegisterHttpHandler'")
	}

	if logger == nil {
		logger = watermill.NopLogger{}
	}

	return &Subscriber{
		config:             config,
		logger:             logger,
		outputChannels:     make([]chan *message.Message, 0),
		outputChannelsLock: &sync.Mutex{},
	}, nil
}

func (c *SubscriberConfig) setDefaults() {
	if c.UnmarshalMessageFunc == nil {
		c.UnmarshalMessageFunc = DefaultUnmarshalMessageFunc
	}
}

// Subscribe creates a HTTP handler which will listen on provided url for messages. The callee must
// register this HTTP handler on it's mux of choice before calling `StartHTTPServer` or equivalent.
//
// When request is sent, it will wait for the `Ack`. When Ack is received 200 HTTP status will be sent.
// When Nack is sent, 500 HTTP status will be sent.
func (s *Subscriber) Subscribe(ctx context.Context, url string) (<-chan *message.Message, error) {

	messages := make(chan *message.Message)

	s.outputChannelsLock.Lock()
	s.outputChannels = append(s.outputChannels, messages)
	s.outputChannelsLock.Unlock()

	baseLogFields := watermill.LogFields{"url": url, "provider": "googlecloud_http"}

	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}

	// Tell the callee to register this http handler on it's mux of choice
	s.config.RegisterHttpHandler(url, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		msg, err := s.config.UnmarshalMessageFunc(r)

		if err != nil {
			s.logger.Info("Cannot unmarshal message", baseLogFields.Add(watermill.LogFields{"err": err}))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if msg == nil {
			s.logger.Info("No message returned by UnmarshalMessageFunc", baseLogFields)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		ctx, cancelCtx := context.WithCancel(ctx)
		msg.SetContext(ctx)
		defer cancelCtx()

		logFields := baseLogFields.Add(watermill.LogFields{"message_uuid": msg.UUID})

		s.logger.Trace("Sending msg", logFields)
		messages <- msg

		s.logger.Trace("Waiting for ACK", logFields)
		select {
		case <-msg.Acked():
			s.logger.Trace("Message acknowledged", logFields.Add(watermill.LogFields{"err": err}))
			w.WriteHeader(http.StatusOK)
		case <-msg.Nacked():
			s.logger.Info("Message nacked", logFields.Add(watermill.LogFields{"err": err}))
			w.WriteHeader(http.StatusInternalServerError)
		case <-r.Context().Done():
			s.logger.Info("Request stopped without ACK received", logFields)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))

	return messages, nil
}

func (s *Subscriber) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true

	for _, ch := range s.outputChannels {
		close(ch)
	}

	return nil
}
