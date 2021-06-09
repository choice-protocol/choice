# choice Operator

A proxy that:

- sends to a standard endpoint (infura/alchemy/etc) everything except transactions (no recording)
- records those transactions it does not send to firestore

## Running
Configure the bidder
```
$ export CHOICE_BIDDER_URL=https://rinkeby-light.eth.linkpool.io
```

Configure the standard vanilla (infura/alchemy/etc) endpoint
```
$ export CHOICE_VANILLA_URL=https://rinkeby-light.eth.linkpool.io
```

Configure the server listening port
```
$ export CHOICE_PORT=8545
```

Run the server
```
$ go run main.go
```
