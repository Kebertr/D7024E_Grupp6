# Start the CLI

To start the client open your terminal and run ``go run ./cmd start --port 8000`` from project root directory.

To check wheter another Kademlia node is reachable use go run ``./cmd ping --ip 127.0.0.1 --port 8000``

## Value size limit

The DHT accepts values up to 4096 bytes (4 KiB). This limit was chosen because
4096-bytes keeps values within a single UDP datagram while allowing moderately sized files to be stored. Values larger than 4096 bytes are rejected.

# Run Tests

To run all test simply open your terminal and run: ``./test.sh `` from project root directory

# Coverage report

To get a Coverage report of all the tests run: ``./coveragereport.sh `` from project root directory


# Running Experiments 

To run experiments run `./run.sh` from project root directory

This might take some minutes to complete, so you might want to get a cup of coffe while this is running.
