A stupid simple pub/sub node. Basically just a place where you can funnel a bunch of events and then it will send an HTTP POST to all of the subscribers.

To start a node run

```bash
go run main example/config.json
```

Your `config.json` should contain

1. A mapping between topics and processes to run. When the subscriber recieves a new message on a topic, it runs the corresponding process with the payload passed as a json file in the first command line argument.
2. A mapping from topics to default subscribers. This can be empty. Each is notified when the node recieves a new event. Note that chains can be created if a subscriber publishes an event.
3. A list of known publishers to subscribe from. Each topic you want should be published by exactly one of them.
4. The address of your node. Start ngrok/nginx/etc... prior to fantasma to get this.


There is no consensus, so only one node can publish on a given topic.
There is no authorization, so this is not secure.
There is no retrying or backoff, so this is not robust.
Everything is serialized in JSON.
There are no data contracts.
All communication happens over HTTP.