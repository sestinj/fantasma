A stupid simple message queueing system. Basically just a place where you can funnel a bunch of events and then it will send an HTTP POST to all of the subscribers.
There are subscriber and publisher nodes.

To start a subscriber node run

```bash
go fantasma subscriber sub_config.json
```

To start a publisher node run

```bash
go fantasma publisher pub_config.json
```

You can also edit the configs while the node is running, just edit the file and it will automatically detect changes.

The `sub_config.json` file is simply a mapping between topics and processes to run. When the subscriber recieves a new message, it will look for the process to run for that topic, and pass the messages contents to an instance of the process. The message is always in JSON format and is passed to the process via a file. This means you can read the message from any programming language.

The `pub_config.json` file simply contains a mapping from topics to subscribers.

If a subscriber process wants to publish to a topic, it can simply make an HTTP call to the publishing node it wishes to use—this should likely be the same subscriber that called it.

In order to set up trust between publisher and subscriber, the subscriber must have a key that allows it access to the publisher's streams. Similarly, a key must be presented in order to post an event to a publisher node.