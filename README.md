# Porter

Воркер для конкурентого выполнения работ с общим состоянием

## Examples

1. [helloworld](/examples/helloworld/main.go) - simple hello world worker
2. [events](/examples/events/main.go) - adding an event handler to the worker
3. [jobttl](/examples/jobttl/main.go) - job lifetime usage
4. [recover](/examples/recover/main.go) - panic handling


## Benchmarks

To run benchmarks use `make bench`

```
cpu: AMD Ryzen 7 3700X 8-Core Processor

Benchmark
Benchmark/ForLoop
Benchmark/ForLoop-16         	 8253277	       169.7 ns/op	       0 B/op	       0 allocs/op
Benchmark/Default
Benchmark/Default-16         	 1000000	      1145 ns/op	     112 B/op	       2 allocs/op
Benchmark/WithJobTTL
Benchmark/WithJobTTL-16      	  170295	      6926 ns/op	     712 B/op	      13 allocs/op
Benchmark/WithJobID
Benchmark/WithJobID-16       	  340582	      3562 ns/op	     272 B/op	       8 allocs/op
```
