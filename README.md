# Squadra Corse PoliTO Telemetry Server

The Squadra Corse PoliTO Telemetry Server (sc-telemetry for short) serves as the processor of the CAN bus telemetry stream coming from our race car.

Its main goal is to collect [Cannelloni](https://github.com/mguentner/cannelloni) encoded UDP packets coming from the [SCanner](https://github.com/squadracorsepolito/SCanner) board, decode them, and store the human-readable data in QuestDB, an high performance time series database.

Under the hood, sc-telemetry uses [goccia](https://github.com/FerroO2000/goccia), a Go library for building high performance data processing pipelines. It uses a 6 stage pipeline, with the following stages:

1. UDP Ingress: receives UDP packets from the SCanner board.
2. Cannelloni Decoder Processor: decodes the UPD payload using the Cannelloni specification. At this point, the CAN messages are still raw.
3. ROB Processor: sorts the received Cannelloni packets based on the the sequence number. This is crucial, because the UDP packets can arrive out of order. In case of out of order packets, the ROB processor will buffer them and send them to the next stage in order. It also adjust the timestamp of the Cannelloni messages.
4. CAN Processor: decodes the raw CAN messages using the spefications contained in the DBC file. Underneath, it uses [acmelib](https://github.com/squadracorsepolito/acmelib) to parse the DBC file and decode the raw CAN messages.
5. CAN Message Handler: applies custom logic to the CAN messages. It is the only _custom_ stage of the pipeline. Basically, it creates the corresponding QuestDB messages based on the CAN messages, e.g. it puts integer signals into the `int_signals` QuestDB table.
6. QuestDB Egress: stores the CAN messages into QuestDB.

## Configuration

The configuration of the sc-telemetry server is contained in a YAML file. By default, the container expects the configuration file to be placed in `/app/config/config.yaml`. This behavior can be changed by setting the `CONFIG_PATH` environment variable, e.g. `CONFIG_PATH=/path/to/config.yaml`. Each configuration field can be overridden by setting the corresponding environment variable, e.g. `CONNECTOR_SIZE=1024`. The environment variables used to override nested configuration fields must be prefixed with the name of the field, e.g. `UDP_PORT=1234`.

This is the default configuration file, also available [here](./config.yaml):

```yaml
# sc-telemetry default configuration

# The service name is used to add resource attributes to
# traces, metrics, and logs
service_name: sc-telemetry

# The connector size is used to configure the ring buffer size
# to use between stages
connector_size: 2048

# UDP ingress stage configuration
udp:
    # The address to listen on
    ip_addr: 127.0.0.1
    # The port to listen on
    port: 20_000

# Cannelloni decoder processor stage configuration
cannelloni:
    # The running mode of the stage (pool or single)
    running_mode: pool
    # The maximum number of workers to use (pool mode only)
    max_workers: 4
    # The target queue depth used for scaling up/down
    # the number of workers (pool mode only)
    target_queue_depth: 64

# ROB (Re-Order Buffer) processor stage configuration
rob:
    # The time duration to wait before resetting the ROB.
    # This is used to prevent the ROB from waiting
    # for messages that are lost
    reset_timeout: 100ms

# CAN processor stage configuration
can:
    # The running mode of the stage (pool or single)
    running_mode: pool
    # The maximum number of workers to use (pool mode only)
    max_workers: 4
    # The target queue depth used for scaling up/down
    # the number of workers (pool mode only)
    target_queue_depth: 64

    # The path to the dbc file to use for decoding
    # CAN messages
    dbc_file_path: /app/can/bus.dbc

# CAN message handler custom processor stage configuration
can_message_handler:
    # The running mode of the stage (pool or single)
    running_mode: pool
    # The maximum number of workers to use (pool mode only)
    max_workers: 4
    # The target queue depth used for scaling up/down
    # the number of workers (pool mode only)
    target_queue_depth: 64

# QuestDB egress stage configuration
quest_db:
    # The running mode of the stage (pool or single)
    running_mode: pool
    # The maximum number of workers to use (pool mode only)
    max_workers: 4
    # The target queue depth used for scaling up/down
    # the number of workers (pool mode only)
    target_queue_depth: 64
    # The address to connect to
    address: http://questdb:9009
```
