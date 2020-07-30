# Svc2Handler
> auto convert your go func or service to http handler

focus on service not server



## Benchmark

```
BenchmarkRun-20                       	 7398498	       161 ns/op	       0 B/op	       0 allocs/op
BenchmarkRunContext-20                	 1878054	       635 ns/op	       0 B/op	       0 allocs/op
BenchmarkRunMap-20                    	 2806616	       428 ns/op	      64 B/op	       3 allocs/op
BenchmarkRunStruct-20                 	 3117616	       386 ns/op	      56 B/op	       3 allocs/op
BenchmarkRunMultiParam-20             	 2031732	       591 ns/op	     104 B/op	       4 allocs/op
BenchmarkRunStructWithCtx-20          	 1000000	      1045 ns/op	      88 B/op	       3 allocs/op
BenchmarkRunMultiParamContext-20      	 1000000	      1172 ns/op	     144 B/op	       4 allocs/op
BenchmarkRunMultiParam2-20            	 1252812	       954 ns/op	     256 B/op	       7 allocs/op
BenchmarkRunMultiStruct5WithCtx-20    	  812223	      1486 ns/op	     288 B/op	       7 allocs/op
BenchmarkRunDef-20                    	261235088	         4.52 ns/op	       0 B/op	       0 allocs/op
```