.PHONY: all

all: build

build: eupho.proto
	protoc --proto_path=../pet:. --go_out=Mpet.proto=gopkg.in/mix3/pet.v2,plugins=grpc:. eupho.proto
