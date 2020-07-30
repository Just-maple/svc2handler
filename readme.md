# Svc2Handler
> auto convert your go func or service to http handler

focus on service not server



## Benchmark

```
BenchmarkRun-20                       	 7284895	       166 ns/op	       0 B/op	       0 allocs/op
BenchmarkRunContext-20                	 1860045	       648 ns/op	       0 B/op	       0 allocs/op
BenchmarkRunMap-20                    	 2746640	       438 ns/op	      64 B/op	       3 allocs/op
BenchmarkRunStruct-20                 	 3104311	       388 ns/op	      56 B/op	       3 allocs/op
BenchmarkRunMultiParam-20             	 2006410	       601 ns/op	     104 B/op	       4 allocs/op
BenchmarkRunStructWithCtx-20          	 1000000	      1061 ns/op	      88 B/op	       3 allocs/op
BenchmarkRunMultiParamContext-20      	 1000000	      1174 ns/op	     144 B/op	       4 allocs/op
BenchmarkRunMultiParam2-20            	 1251198	       970 ns/op	     256 B/op	       7 allocs/op
BenchmarkRunMultiStruct5WithCtx-20    	  802453	      1523 ns/op	     288 B/op	       7 allocs/op
BenchmarkRunDef-20                    	262432466	         4.58 ns/op	       0 B/op	       0 allocs/op
```