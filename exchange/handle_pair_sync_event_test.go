package exchange

import (
	"io/ioutil"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/test-go/testify/require"
)

func TestHandlePairSyncEvent(t *testing.T) {
	t.Skipf("ski pair sync event")
	testCase := &TestCase{}
	pairSyncEventYaml, err := ioutil.ReadFile("./testdata/TestHandlePairSyncEvent.yaml")
	require.NoError(t, err)

	err = yaml.Unmarshal(pairSyncEventYaml, &testCase)
	require.NoError(t, err)

	intrinsics := NewTestIntrinsics(testCase)
	sg := NewTestSubgraph(intrinsics)

	err = sg.Init()
	require.NoError(t, err)

	for _, ev := range testCase.Events {
		err := sg.HandleEvent(ev.Event)
		if err != nil {
			t.Errorf(err.Error())
		}
	}
}

func TestGetBNBPriceInUSD_BUSDOnly(t *testing.T) {
	t.Skipf("ski pair sync event")
	testCase := &TestCase{}
	storeYaml := []byte(`---
storeData:
  - type: pair
    entity:
      id: "0x58f876857a02d6762e0101bb5c46a8c1ed44dc16"
      name: "BUSD-WBNB"
      token1Price: "10.00"
      reserve0: "100"
`)

	err := yaml.Unmarshal(storeYaml, &testCase)
	require.NoError(t, err)

	intrinsics := NewTestIntrinsics(testCase)
	sg := NewTestSubgraph(intrinsics)

	err = sg.Init()
	require.NoError(t, err)

	res, err := sg.GetBnbPriceInUSD()
	require.NoError(t, err)

	resFloat, _ := res.Float64()

	require.InEpsilon(t, resFloat, 10.00, 0.0001)
}

func TestGetBNBPriceInUSD_USDTOnly(t *testing.T) {
	t.Skipf("ski pair sync event")
	testCase := &TestCase{}
	storeYaml := []byte(`---
storeData:
  - type: pair
    entity:
      id: "0x16b9a82891338f9ba80e2d6970fdda79d1eb0dae"
      name: "USDT-WBNB"
      token0Price: "5.00"
      reserve1: "50"
`)

	err := yaml.Unmarshal(storeYaml, &testCase)
	require.NoError(t, err)

	intrinsics := NewTestIntrinsics(testCase)
	sg := NewTestSubgraph(intrinsics)

	err = sg.Init()
	require.NoError(t, err)

	res, err := sg.GetBnbPriceInUSD()
	require.NoError(t, err)

	resFloat, _ := res.Float64()

	require.InEpsilon(t, resFloat, 5.00, 0.0001)
}

func TestGetBNBPriceInUSD_BothPairsExist(t *testing.T) {
	t.Skipf("ski pair sync event")
	testCase := &TestCase{}
	storeYaml := []byte(`---
storeData:
  - type: pair
    entity:
      id: "0x58f876857a02d6762e0101bb5c46a8c1ed44dc16"
      name: "BUSD-WBNB"
      token1Price: "10.00"
      reserve0: "100"
  - type: pair
    entity:
      id: "0x16b9a82891338f9ba80e2d6970fdda79d1eb0dae"
      name: "USDT-WBNB"
      token0Price: "5.00"
      reserve1: "50"
`)

	err := yaml.Unmarshal(storeYaml, &testCase)
	require.NoError(t, err)

	intrinsics := NewTestIntrinsics(testCase)
	sg := NewTestSubgraph(intrinsics)

	err = sg.Init()
	require.NoError(t, err)

	res, err := sg.GetBnbPriceInUSD()
	require.NoError(t, err)

	resFloat, _ := res.Float64()

	require.InEpsilon(t, resFloat, 8.333333, 0.0001)
}

func TestGetBNBPriceInUSD_BothPairsExist_ZeroLiquidity(t *testing.T) {
	t.Skipf("ski pair sync event")
	testCase := &TestCase{}
	storeYaml := []byte(`---
storeData:
  - type: pair
    entity:
      id: "0x58f876857a02d6762e0101bb5c46a8c1ed44dc16"
      name: "BUSD-WBNB"
      token1Price: "10.00"
      reserve0: "0"
  - type: pair
    entity:
      id: "0x16b9a82891338f9ba80e2d6970fdda79d1eb0dae"
      name: "USDT-WBNB"
      token0Price: "5.00"
      reserve1: "0"
`)

	err := yaml.Unmarshal(storeYaml, &testCase)
	require.NoError(t, err)

	intrinsics := NewTestIntrinsics(testCase)
	sg := NewTestSubgraph(intrinsics)

	err = sg.Init()
	require.NoError(t, err)

	res, err := sg.GetBnbPriceInUSD()
	require.NoError(t, err)

	resFloat, _ := res.Float64()
	require.Zero(t, resFloat)
}
