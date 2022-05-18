# Watermill Google Cloud Pub/Sub Http Push

Provides support for http push subscriptions, as an alternative to the official [watermill-googlecloud](https://github.com/ThreeDotsLabs/watermill-googlecloud) repository. It was created to support our services running on Cloud Run, which restricts us to use push subscriptions on Google Cloud.

So this is in other words some kind of hybrid of [watermill-googlecloud](https://github.com/ThreeDotsLabs/watermill-googlecloud) and [watermill-http](https://github.com/ThreeDotsLabs/watermill-http), but does not enforce the use of chi and let's you be "in charge" of the http server.

At Dentech, we use this library together with our custom [dentech-floss/watermill-opentelemetry-go-extra](https://github.com/dentech-floss/watermill-opentelemetry-go-extra) lib to get Opentelemetry traces propagated from publishers to subscribers (child span).

## Usage

### For subscribers

This example assume that a push subscription which points to the service has been created, then you provide a func that shall register this libs http handler on the url that the messages will be pushed to.

We start with the setup of the http push subscriber:

```go
package example

import (
    "github.com/ThreeDotsLabs/watermill/message"
    "github.com/garsue/watermillzap"

    googlecloud_http "github.com/dentech-floss/watermill-googlecloud-http/pkg/googlecloud/http"
)

func NewSubscriber(
    logger *zap.Logger, 
    registerHttpHandler googlecloud_http.RegisterHttpHandler,
) (message.Subscriber, error) {

    subscriber, err := googlecloud_http.NewSubscriber(
        googlecloud_http.SubscriberConfig{
            RegisterHttpHandler: registerHttpHandler,
        },
        watermillzap.NewLogger(logger),
    )
    if err != nil {
        return nil, err
    }

    return subscriber
}
```

...which we can make use of something like this for example:

```go
package example

import (
    "github.com/ThreeDotsLabs/watermill/message"

    "github.com/go-chi/chi"
)

func main() {
    httpRouter := chi.NewRouter() // not limited to chi...

    subscriber := NewSubscriber(logger, httpRouter.Handle)
    // ...router definition omitted for clarity
    watermillRouter.AddNoPublisherHandler(
        "pubsub.Subscribe",
        "/push-handlers/pubsub/test", // the topic/url we get messages pushed to from PubSub
        subscriber,
        func(msg *message.Message) error {
            // ...do something with the message...
            msg.Ack()
            return nil
        },
    )
}

```
