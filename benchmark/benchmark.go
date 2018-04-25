package main

import (
	"flag"
	"log"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"errors"

	"github.com/chuangyou/qsf/breaker"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/montanaflynn/stats"
	"github.com/rubyist/circuitbreaker"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	grpcInitialWindowSize     = 1 << 30
	grpcInitialConnWindowSize = 1 << 30
	MaxSendMsgSize            = 1<<31 - 1
	MaxCallMsgSize            = 1<<31 - 1
)

var concurrency = flag.Int("c", 1, "concurrency")
var total = flag.Int("n", 1, "total requests for all clients")
var host = flag.String("s", "127.0.0.1:50055", "server ip and port")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()
	conc, tn, err := checkArgs(*concurrency, *total)
	if err != nil {
		log.Printf("err: %v", err)
		return
	}
	n := conc
	m := tn / n
	log.Printf("concurrency: %d\nrequests per client: %d\n\n", n, m)

	var wg sync.WaitGroup
	wg.Add(n * m)

	log.Printf("sent total %d messages, %d message per client", n*m, m)
	var startWg sync.WaitGroup
	startWg.Add(n)

	var trans uint64
	var transOK uint64

	d := make([][]int64, n, n)

	//it contains warmup time but we can ignore it
	totalT := time.Now().UnixNano()
	for i := 0; i < n; i++ {
		dt := make([]int64, 0, m)
		d = append(d, dt)

		go func(i int) {
			defer func() {
				if r := recover(); r != nil {
					log.Print("Recovered in f", r)
				}
			}()
			//breaker
			breaker := circuit.NewRateBreaker(0.75, 100)
			//breaker
			grpOpts := []grpc.DialOption{
				grpc.WithInsecure(),
				grpc.WithInitialWindowSize(grpcInitialWindowSize),
				grpc.WithInitialConnWindowSize(grpcInitialConnWindowSize),
				grpc.WithDefaultCallOptions(
					grpc.MaxCallRecvMsgSize(MaxCallMsgSize),
					grpc.MaxCallSendMsgSize(MaxSendMsgSize),
				),

				grpc.WithUnaryInterceptor(
					grpc_middleware.ChainUnaryClient(
						grpc_breaker.UnaryClientInterceptor(breaker),
						//otgrpc.OpenTracingClientInterceptor(tracer),
					),
				),
				grpc.WithPerRPCCredentials(new(exampleServiceCredential)),
			}
			conn, _ := grpc.Dial(*host, grpOpts...)

			defer conn.Close()
			xclient := NewExampleServiceClient(conn)
			//warmup
			for j := 0; j < 5; j++ {
				xclient.GetExample(context.Background(), &GetExampleRequest{Value: "test"})
			}

			startWg.Done()
			startWg.Wait()

			for j := 0; j < m; j++ {
				t := time.Now().UnixNano()
				_, err := xclient.GetExample(context.Background(), &GetExampleRequest{Value: "test" + strconv.Itoa(j)})

				t = time.Now().UnixNano() - t

				d[i] = append(d[i], t)

				if err == nil {
					atomic.AddUint64(&transOK, 1)
				}

				if err != nil {
					log.Print(err.Error())
				}

				atomic.AddUint64(&trans, 1)
				wg.Done()
			}
		}(i)

	}

	wg.Wait()

	totalT = time.Now().UnixNano() - totalT
	log.Printf("took %f ms for %d requests\n", float64(totalT)/1000000, n*m)

	totalD := make([]int64, 0, n*m)
	for _, k := range d {
		totalD = append(totalD, k...)
	}
	totalD2 := make([]float64, 0, n*m)
	for _, k := range totalD {
		totalD2 = append(totalD2, float64(k))
	}

	mean, _ := stats.Mean(totalD2)
	median, _ := stats.Median(totalD2)
	max, _ := stats.Max(totalD2)
	min, _ := stats.Min(totalD2)
	p99, _ := stats.Percentile(totalD2, 99.9)

	log.Printf("sent     requests    : %d\n", n*m)
	log.Printf("received requests    : %d\n", atomic.LoadUint64(&trans))
	log.Printf("received requests_OK : %d\n", atomic.LoadUint64(&transOK))
	log.Printf("throughput  (TPS)    : %d\n", int64(n*m)*1000000000/totalT)
	log.Printf("mean: %.f ns, median: %.f ns, max: %.f ns, min: %.f ns, p99.9: %.f ns\n", mean, median, max, min, p99)
	log.Printf("mean: %d ms, median: %d ms, max: %d ms, min: %d ms, p99: %d ms\n", int64(mean/1000000), int64(median/1000000), int64(max/1000000), int64(min/1000000), int64(p99/1000000))
}

type exampleServiceCredential struct{}

func (c exampleServiceCredential) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "basic GET",
	}, nil
}
func (c exampleServiceCredential) RequireTransportSecurity() bool {

	return false
}

// checkArgs check concurrency and total request count.
func checkArgs(c, n int) (int, int, error) {
	if c < 1 {
		log.Printf("c < 1 and reset c = 1")
		c = 1
	}
	if n < 1 {
		log.Printf("n < 1 and reset n = 1")
		n = 1
	}
	if c > n {
		return c, n, errors.New("c must be set <= n")
	}
	return c, n, nil
}
