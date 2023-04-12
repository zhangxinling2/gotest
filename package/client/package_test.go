package client

import (
	"gotest/package/series"
	"testing"
)

func TestPackage(t *testing.T) {
	series.GetFibonacci(2)
}
