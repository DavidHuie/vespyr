package vespyr_test

import (
	"testing"

	"github.com/DavidHuie/vespyr/pkg/vespyr"
	"github.com/stretchr/testify/assert"
)

func TestTruncateFloat(t *testing.T) {
	cases := []struct {
		arg, response float64
		precision     uint
	}{
		{49.760686138506, 49.76068613, 8},
		{49.7606, 49.7606, 8},
		{49.7606, 49.7, 1},
		{49.7606, 49, 0},
		{-49.7606, -49, 0},
		{-49.7606, -49.7, 1},
	}

	for _, c := range cases {
		assert.Equal(t, c.response, vespyr.TruncateFloat(c.arg, c.precision))
	}
}
