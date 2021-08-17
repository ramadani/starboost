# starboost

Kafka Event Client Tool

- Publish message via Rest API
- Configurable consume message

## Installation

1. Download zip file on [release](https://github.com/ramadani/starboost/releases) page then extract it
2. Create yaml config file based on example below

**Example Config**

```yaml
# config.yaml
address: localhost:8123
debug: false
kafka:
  addresses:
    - localhost:9092
consumer:
  enabled: true
  groupId: starboost
  topics:
    - paybill
    - tap_snap
  output:
    stdout: true
    file: starboost.log
```

## Usage

Run starboost based on your platform

```bash
$ ./starboost-{your-platform} --config=config.yaml
```

### Publish

Publish message via endpoint `POST /publish`. You can use cURL or Rest Client.

Example 1

```bash
curl --location --request POST 'localhost:8123/publish' \
--header 'Content-Type: application/json' \
--data-raw '{
    "topic": "paybill",
    "message": "some message"
}'
```

Example 2

```bash
curl --location --request POST 'localhost:8123/publish' \
--header 'Content-Type: application/json' \
--data-raw '{
    "topic": "paybill",
    "message": {
        "foo": "bar"
    }
}'
```

### Consume

You can see the incoming messages on your terminal or log file based on consumer output config.

**Example Config**

```yaml
consumer:
  ...
  output:
    stdout: true
    file: starboost.log
```

## TODO

- [ ] Filterable on consuming messages
- [ ] Dockerize

## Contributing

1. Fork the Project
2. Create your Feature Branch (git checkout -b feature/AmazingFeature)
3. Commit your Changes (git commit -m 'Add some AmazingFeature')
4. Push to the Branch (git push origin feature/AmazingFeature)
5. Open a Pull Request

## License

Distributed under the MIT License. See `LICENSE` for more information.
