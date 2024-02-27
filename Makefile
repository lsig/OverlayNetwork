
generate:
	protoc --proto_path=. --go_out=./pb --go_opt=paths=source_relative minichord.proto
