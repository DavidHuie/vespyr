package vespyr_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/DavidHuie/vespyr/pkg/vespyr"
	"github.com/stretchr/testify/assert"
)

func TestCandlestickBuilder(t *testing.T) {
	startTime := time.Now().Add(-time.Minute)
	endTime := time.Now()

	builder := vespyr.NewCandlestickBuilder(vespyr.ProductBTCUSD, startTime, endTime)
	builder.ProcessMessage(&vespyr.ExchangeMessage{
		Type:        string(vespyr.MessageMatch),
		Price:       2800,
		Size:        1,
		ProductType: string(vespyr.ProductBTCUSD),
		Time:        startTime.Add(time.Second),
	})

	candlestick := builder.Build()

	assert.Equal(t, startTime, candlestick.StartTime)
	assert.Equal(t, endTime, candlestick.EndTime)
	assert.Equal(t, float64(2800), candlestick.Low)
	assert.Equal(t, float64(2800), candlestick.High)
	assert.Equal(t, float64(2800), candlestick.Open)
	assert.Equal(t, float64(2800), candlestick.Close)
	assert.Equal(t, float64(1), candlestick.Volume)
	assert.Equal(t, vespyr.CandlestickDirectionUp, candlestick.Direction)
	assert.Equal(t, vespyr.ProductBTCUSD, candlestick.Product)
}

func TestCandlestickBuilderUp(t *testing.T) {
	startTime := time.Now().Add(-time.Minute)
	endTime := time.Now()

	builder := vespyr.NewCandlestickBuilder(vespyr.ProductBTCUSD, startTime, endTime)

	builder.ProcessMessage(&vespyr.ExchangeMessage{
		Type:        string(vespyr.MessageMatch),
		Price:       2800,
		Size:        1,
		ProductType: string(vespyr.ProductBTCUSD),
		Time:        startTime.Add(time.Second),
	})
	builder.ProcessMessage(&vespyr.ExchangeMessage{
		Type:        string(vespyr.MessageMatch),
		Price:       2900,
		Size:        3,
		ProductType: string(vespyr.ProductBTCUSD),
		Time:        startTime.Add(time.Second),
	})

	candlestick := builder.Build()

	assert.Equal(t, startTime, candlestick.StartTime)
	assert.Equal(t, endTime, candlestick.EndTime)
	assert.Equal(t, float64(2800), candlestick.Low)
	assert.Equal(t, float64(2900), candlestick.High)
	assert.Equal(t, float64(2800), candlestick.Open)
	assert.Equal(t, float64(2900), candlestick.Close)
	assert.Equal(t, float64(4), candlestick.Volume)
	assert.Equal(t, vespyr.CandlestickDirectionUp, candlestick.Direction)
	assert.Equal(t, vespyr.ProductBTCUSD, candlestick.Product)
}

func TestCandlestickBuilderDown(t *testing.T) {
	startTime := time.Now().Add(-time.Minute)
	endTime := time.Now()

	builder := vespyr.NewCandlestickBuilder(vespyr.ProductBTCUSD, startTime, endTime)

	builder.ProcessMessage(&vespyr.ExchangeMessage{
		Type:        string(vespyr.MessageMatch),
		Price:       2800,
		Size:        1,
		ProductType: string(vespyr.ProductBTCUSD),
		Time:        startTime.Add(time.Second),
	})
	builder.ProcessMessage(&vespyr.ExchangeMessage{
		Type:        string(vespyr.MessageMatch),
		Price:       2700,
		Size:        3,
		ProductType: string(vespyr.ProductBTCUSD),
		Time:        startTime.Add(time.Second),
	})

	candlestick := builder.Build()

	assert.Equal(t, startTime, candlestick.StartTime)
	assert.Equal(t, endTime, candlestick.EndTime)
	assert.Equal(t, float64(2700), candlestick.Low)
	assert.Equal(t, float64(2800), candlestick.High)
	assert.Equal(t, float64(2800), candlestick.Open)
	assert.Equal(t, float64(2700), candlestick.Close)
	assert.Equal(t, float64(4), candlestick.Volume)
	assert.Equal(t, vespyr.CandlestickDirectionDown, candlestick.Direction)
	assert.Equal(t, vespyr.ProductBTCUSD, candlestick.Product)
}

func TestCandlestickBuilderIgnore(t *testing.T) {
	startTime := time.Now().Add(-time.Minute)
	endTime := time.Now()

	builder := vespyr.NewCandlestickBuilder(vespyr.ProductBTCUSD, startTime, endTime)

	builder.ProcessMessage(&vespyr.ExchangeMessage{
		Type:        string(vespyr.MessageMatch),
		Price:       2800,
		Size:        1,
		ProductType: string(vespyr.ProductBTCUSD),
		Time:        startTime.Add(time.Second),
	})

	candlestick := builder.Build()

	assert.Equal(t, startTime, candlestick.StartTime)
	assert.Equal(t, endTime, candlestick.EndTime)
	assert.Equal(t, float64(2800), candlestick.Low)
	assert.Equal(t, float64(2800), candlestick.High)
	assert.Equal(t, float64(2800), candlestick.Open)
	assert.Equal(t, float64(2800), candlestick.Close)
	assert.Equal(t, float64(1), candlestick.Volume)
	assert.Equal(t, vespyr.CandlestickDirectionUp, candlestick.Direction)
	assert.Equal(t, vespyr.ProductBTCUSD, candlestick.Product)

	// Time
	builder.ProcessMessage(&vespyr.ExchangeMessage{
		Type:        string(vespyr.MessageMatch),
		Price:       2700,
		Size:        1,
		ProductType: string(vespyr.ProductBTCUSD),
		Time:        startTime.Add(10 * time.Minute),
	})

	candlestick = builder.Build()

	assert.Equal(t, startTime, candlestick.StartTime)
	assert.Equal(t, endTime, candlestick.EndTime)
	assert.Equal(t, float64(2800), candlestick.Low)
	assert.Equal(t, float64(2800), candlestick.High)
	assert.Equal(t, float64(2800), candlestick.Open)
	assert.Equal(t, float64(2800), candlestick.Close)
	assert.Equal(t, float64(1), candlestick.Volume)
	assert.Equal(t, vespyr.CandlestickDirectionUp, candlestick.Direction)
	assert.Equal(t, vespyr.ProductBTCUSD, candlestick.Product)

	// Message type
	builder.ProcessMessage(&vespyr.ExchangeMessage{
		Type:        string("buy"),
		Price:       2900,
		Size:        1,
		ProductType: string(vespyr.ProductBTCUSD),
	})

	candlestick = builder.Build()

	assert.Equal(t, startTime, candlestick.StartTime)
	assert.Equal(t, endTime, candlestick.EndTime)
	assert.Equal(t, float64(2800), candlestick.Low)
	assert.Equal(t, float64(2800), candlestick.High)
	assert.Equal(t, float64(2800), candlestick.Open)
	assert.Equal(t, float64(2800), candlestick.Close)
	assert.Equal(t, float64(1), candlestick.Volume)
	assert.Equal(t, vespyr.CandlestickDirectionUp, candlestick.Direction)
	assert.Equal(t, vespyr.ProductBTCUSD, candlestick.Product)

	// Product type
	builder.ProcessMessage(&vespyr.ExchangeMessage{
		Type:        string(vespyr.MessageMatch),
		Price:       3000,
		Size:        1,
		ProductType: "ETH-USD",
	})

	candlestick = builder.Build()

	assert.Equal(t, startTime, candlestick.StartTime)
	assert.Equal(t, endTime, candlestick.EndTime)
	assert.Equal(t, float64(2800), candlestick.Low)
	assert.Equal(t, float64(2800), candlestick.High)
	assert.Equal(t, float64(2800), candlestick.Open)
	assert.Equal(t, float64(2800), candlestick.Close)
	assert.Equal(t, float64(1), candlestick.Volume)
	assert.Equal(t, vespyr.CandlestickDirectionUp, candlestick.Direction)
	assert.Equal(t, vespyr.ProductBTCUSD, candlestick.Product)
}

func TestCandlestickBuilderCandlestick(t *testing.T) {
	startTime := time.Now().Add(-time.Minute)
	endTime := time.Now()

	builder := vespyr.NewCandlestickBuilder(vespyr.ProductBTCUSD, startTime, endTime)

	builder.ProcessCandlestickModel(&vespyr.CandlestickModel{
		Product:   vespyr.ProductBTCUSD,
		StartTime: startTime.Add(time.Second),
		EndTime:   startTime.Add(time.Minute),
		Low:       2,
		High:      5,
		Open:      1,
		Close:     4,
		Volume:    10,
	})

	candlestick := builder.Build()

	assert.Equal(t, startTime, candlestick.StartTime)
	assert.Equal(t, endTime, candlestick.EndTime)
	assert.Equal(t, float64(2), candlestick.Low)
	assert.Equal(t, float64(5), candlestick.High)
	assert.Equal(t, float64(1), candlestick.Open)
	assert.Equal(t, float64(4), candlestick.Close)
	assert.Equal(t, float64(10), candlestick.Volume)
	assert.Equal(t, vespyr.CandlestickDirectionUp, candlestick.Direction)
	assert.Equal(t, vespyr.ProductBTCUSD, candlestick.Product)

	// Time range
	builder.ProcessCandlestickModel(&vespyr.CandlestickModel{
		Product:   vespyr.ProductBTCUSD,
		StartTime: startTime.Add(time.Minute),
		EndTime:   startTime.Add(2 * time.Minute),
		Low:       2,
		High:      5,
		Open:      1,
		Close:     4,
		Volume:    10,
	})

	candlestick = builder.Build()
	assert.Equal(t, startTime, candlestick.StartTime)
	assert.Equal(t, endTime, candlestick.EndTime)
	assert.Equal(t, float64(2), candlestick.Low)
	assert.Equal(t, float64(5), candlestick.High)
	assert.Equal(t, float64(1), candlestick.Open)
	assert.Equal(t, float64(4), candlestick.Close)
	assert.Equal(t, float64(10), candlestick.Volume)
	assert.Equal(t, vespyr.CandlestickDirectionUp, candlestick.Direction)
	assert.Equal(t, vespyr.ProductBTCUSD, candlestick.Product)

	// Time range
	builder.ProcessCandlestickModel(&vespyr.CandlestickModel{
		Product:   vespyr.ProductBTCUSD,
		StartTime: startTime.Add(-time.Minute),
		EndTime:   startTime.Add(2 * time.Minute),
		Low:       2,
		High:      5,
		Open:      1,
		Close:     4,
		Volume:    10,
	})

	candlestick = builder.Build()
	assert.Equal(t, startTime, candlestick.StartTime)
	assert.Equal(t, endTime, candlestick.EndTime)
	assert.Equal(t, float64(2), candlestick.Low)
	assert.Equal(t, float64(5), candlestick.High)
	assert.Equal(t, float64(1), candlestick.Open)
	assert.Equal(t, float64(4), candlestick.Close)
	assert.Equal(t, float64(10), candlestick.Volume)
	assert.Equal(t, vespyr.CandlestickDirectionUp, candlestick.Direction)
	assert.Equal(t, vespyr.ProductBTCUSD, candlestick.Product)
}

func TestCandlestickBucket(t *testing.T) {
	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		t time.Time
		g int64
		r time.Time
	}{
		{
			time.Unix(1223424000, 0).UTC(),
			1,
			time.Unix(1223424000, 0).UTC(),
		},
		{
			time.Unix(1223424000+15*60, 0).UTC(),
			15,
			time.Unix(1223424000+15*60, 0).UTC(),
		},
		{
			time.Date(2017, time.January, 1, 1, 1, 0, 0, location),
			1,
			time.Date(2017, time.January, 1, 9, 1, 0, 0, time.UTC),
		},
		{
			time.Date(2017, time.January, 1, 1, 0, 0, 0, location),
			15,
			time.Date(2017, time.January, 1, 9, 0, 0, 0, time.UTC),
		},
		{
			time.Date(2017, time.January, 1, 1, 0, 0, 0, location),
			30,
			time.Date(2017, time.January, 1, 9, 0, 0, 0, time.UTC),
		},
		{
			time.Date(2017, time.January, 1, 1, 0, 0, 0, location),
			60,
			time.Date(2017, time.January, 1, 9, 0, 0, 0, time.UTC),
		},
		{
			time.Date(2017, time.January, 1, 1, 0, 0, 0, time.UTC),
			17,
			time.Date(2017, time.January, 1, 0, 58, 0, 0, time.UTC),
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%#v", c), func(t *testing.T) {
			projection := vespyr.CandlestickBucket(c.t, c.g)
			assert.Equal(t, c.r, projection.UTC())
		})
	}
}

func TestReprojectCandlesticks(t *testing.T) {
	startTime := time.Date(2017, time.January, 1, 1, 0, 0, 0, time.Local)

	candles := []*vespyr.CandlestickModel{
		&vespyr.CandlestickModel{
			StartTime: startTime,
			EndTime:   startTime.Add(time.Minute),
			Low:       1,
			High:      10,
			Open:      3,
			Close:     9,
			Volume:    11,
			Product:   vespyr.ProductBTCUSD,
		},
		&vespyr.CandlestickModel{
			StartTime: startTime.Add(time.Minute),
			EndTime:   startTime.Add(2 * time.Minute),
			Low:       1,
			High:      20,
			Open:      2,
			Close:     5,
			Volume:    21,
			Product:   vespyr.ProductBTCUSD,
		},
		&vespyr.CandlestickModel{
			StartTime: startTime.Add(19 * time.Minute),
			EndTime:   startTime.Add(20 * time.Minute),
			Low:       1,
			High:      10,
			Open:      2,
			Close:     9,
			Volume:    11,
			Product:   vespyr.ProductBTCUSD,
		},
		&vespyr.CandlestickModel{
			StartTime: startTime.Add(21 * time.Minute),
			EndTime:   startTime.Add(22 * time.Minute),
			Low:       1,
			High:      15,
			Open:      2,
			Close:     5,
			Volume:    100,
			Product:   vespyr.ProductBTCUSD,
		},
		&vespyr.CandlestickModel{
			StartTime: startTime.Add(30 * time.Minute),
			EndTime:   startTime.Add(31 * time.Minute),
			Low:       1,
			High:      15,
			Open:      2,
			Close:     5,
			Volume:    100,
			Product:   vespyr.ProductBTCUSD,
		},
	}

	projection, err := vespyr.ReprojectCandlesticks(candles, vespyr.ProductBTCUSD, 15)

	assert.NoError(t, err)
	assert.Equal(t, 3, len(projection))

	c1 := projection[0]
	assert.Equal(t, startTime, c1.StartTime)
	assert.Equal(t, startTime.Add(time.Minute*time.Duration(15)), c1.EndTime)
	assert.Equal(t, float64(1), c1.Low)
	assert.Equal(t, float64(20), c1.High)
	assert.Equal(t, float64(3), c1.Open)
	assert.Equal(t, float64(5), c1.Close)
	assert.Equal(t, float64(32), c1.Volume)
	assert.Equal(t, vespyr.CandlestickDirectionUp, c1.Direction)

	c2 := projection[1]
	assert.Equal(t, startTime.Add(time.Minute*time.Duration(15)), c2.StartTime)
	assert.Equal(t, startTime.Add(time.Minute*time.Duration(2*15)), c2.EndTime)
	assert.Equal(t, float64(1), c2.Low)
	assert.Equal(t, float64(15), c2.High)
	assert.Equal(t, float64(2), c2.Open)
	assert.Equal(t, float64(5), c2.Close)
	assert.Equal(t, float64(111), c2.Volume)
	assert.Equal(t, vespyr.CandlestickDirectionUp, c2.Direction)

	c3 := projection[2]
	assert.Equal(t, startTime.Add(time.Minute*time.Duration(30)), c3.StartTime)
	assert.Equal(t, startTime.Add(time.Minute*time.Duration(45)), c3.EndTime)
	assert.Equal(t, float64(1), c3.Low)
	assert.Equal(t, float64(15), c3.High)
	assert.Equal(t, float64(2), c3.Open)
	assert.Equal(t, float64(5), c3.Close)
	assert.Equal(t, float64(100), c3.Volume)
	assert.Equal(t, vespyr.CandlestickDirectionUp, c3.Direction)
}

func TestValidateCandlesticks(t *testing.T) {
	startTime := time.Now()

	values := []*vespyr.CandlestickModel{
		&vespyr.CandlestickModel{
			StartTime: startTime,
		},
		&vespyr.CandlestickModel{
			StartTime: startTime.Add(time.Second * 60),
		},
	}

	assert.NoError(t, vespyr.ValidateCandlesticks(values, 1))

	values = []*vespyr.CandlestickModel{
		&vespyr.CandlestickModel{
			StartTime: startTime,
		},
		&vespyr.CandlestickModel{
			StartTime: startTime.Add(time.Second * 120),
		},
	}

	assert.EqualError(t, vespyr.ValidateCandlesticks(values, 1),
		vespyr.ErrMissingCandlestick.Error())
}
