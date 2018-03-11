google_deps = Mgoogle/protobuf/empty.proto=github.com/golang/protobuf/ptypes/empty
helm_deps = Mhapi/chart/config.proto=k8s.io/helm/pkg/proto/hapi/chart,Mhapi/chart/chart.proto=k8s.io/helm/pkg/proto/hapi/chart
includes = $(HOME)/src/protobuf/include:../../vendor/k8s.io/helm/_proto
plugins = grpc
target = go
deps = $(google_deps),$(helm_deps)
dst = .

.PHONY: rpc
rpc:
	PATH=../../bin:$(PATH) protoc -I=$(includes):. --$(target)_out=plugins=$(plugins),$(deps):$(dst) *.proto

.PHONY: clean
clean:
	rm *.pb.go


