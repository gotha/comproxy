.PHONY: build clean deploy 

build: clean
	export GO111MODULE=on
	go build -o bin/comproxy github.com/gotha/comproxy/cmd/server 	

clean:
	rm -rf ./bin 

run: 
	LOG_LEVEL="DEBUG" \
	  LOG_RESPONSE_BODY=false \
	  APP_PORT=9999 \
	  PROD_SERVICE_URL="http://localhost:8081" \
	  CANDIDATE_SERVICE_URL="http://localhost:8082" \
	  ./bin/comproxy

brun: build run
