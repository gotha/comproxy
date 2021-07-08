# Comproxy

A proxy that compares responses from different services.

Inspired by [diffy](https://github.com/twitter/diffy) this project aims to be much simpler alternative. 

There is no UI, put your logs in something like Splunk and setup alarms if that is your use case.

### Build

```
make build
```

### How to run

```
LOG_LEVEL=DEBUG \
  PROD_SERVICE_URL="http://localhost:8081" \
  CANDIDATE_SERVICE_URL="http://localhost:8082" \
  ./bin/comproxy
```

