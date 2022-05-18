package googlecloud_http

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	pubsub_rest "google.golang.org/api/pubsub/v1"

	"github.com/ThreeDotsLabs/watermill/message"
)

const UUIDHeaderKey = "_watermill_message_uuid"

type UnmarshalMessageFunc func(request *http.Request) (*message.Message, error)

type pushRequest struct {
	PubsubMessage pubsub_rest.PubsubMessage `json:"message"`
	Subscription  string                    `json:"subscription"`
}

func DefaultUnmarshalMessageFunc(r *http.Request) (*message.Message, error) {

	var pushRequest pushRequest
	if err := json.NewDecoder(r.Body).Decode(&pushRequest); err != nil {
		return nil, err
	}

	pubsubMessage := pushRequest.PubsubMessage
	metadata := make(message.Metadata, len(pubsubMessage.Attributes))

	var id string
	for k, attr := range pubsubMessage.Attributes {
		if k == UUIDHeaderKey {
			id = attr
			continue
		}
		metadata.Set(k, attr)
	}

	metadata.Set("publishTime", pubsubMessage.PublishTime)

	payload, err := base64.StdEncoding.DecodeString(pubsubMessage.Data)
	if err != nil {
		return nil, err
	}

	msg := message.NewMessage(id, payload)
	msg.Metadata = metadata

	return msg, nil
}
