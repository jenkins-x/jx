google_deps = Mgoogle/protobuf/timestamp.proto=github.com/golang/protobuf/ptypes/timestamp
includes = ../../vendor/protobuf-include/include/
target = go
plugins =
deps = $(google_deps)
dst = .

.PHONY: proto
proto:
	protoc -I=$(includes):. --$(target)_out=plugins=$(plugins),$(deps):$(dst) *.proto

.PHONY: clean
clean:
	rm *.pb.go
