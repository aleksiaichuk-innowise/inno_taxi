дд# gRPC reading list — before starting Order Service's gRPC work

Read in this order.

1. [Introduction to gRPC](https://grpc.io/docs/what-is-grpc/introduction/)
   What gRPC is, client/server/stub model. Start here.

2. [Core concepts, architecture and lifecycle](https://grpc.io/docs/what-is-grpc/core-concepts/)
   RPC lifecycle, the four RPC types (we only need unary for `CreateOrder`), protobuf as IDL.

3. [Metadata | gRPC](https://grpc.io/docs/guides/metadata/)
   How headers work in gRPC (`metadata.FromIncomingContext`) — the replacement for Gin's `c.GetHeader(...)`.

4. [Authentication | gRPC](https://grpc.io/docs/guides/auth/)
   Where interceptor-based auth fits in gRPC's model.

5. [grpc-go interceptor examples (official repo)](https://github.com/grpc/grpc-go/tree/master/examples/features/interceptor)
   Real Go code: unary interceptor structure, chaining, registration via `grpc.UnaryInterceptor(...)`.

6. [gRPC in Go: Streaming RPCs, Interceptors, and Metadata](https://victoriametrics.com/blog/go-grpc-basic-streaming-interceptor/)
   Practical walkthrough tying interceptors + metadata together in Go.

7. [Customizing your gateway | gRPC-Gateway](https://grpc-ecosystem.github.io/grpc-gateway/docs/mapping/customizing_your_gateway/)
   Header mapping rules — importantly, `Authorization` is special-cased and always forwarded into gRPC metadata, can't be removed by a custom header matcher. Still verify this ourselves end-to-end when we build the HTTP-gateway task.

8. [An all-in-one guide to gRPC-Gateway](https://blog.logrocket.com/guide-to-grpc-gateway/)
   Broader practical guide to gRPC-Gateway as a wrap-up.
