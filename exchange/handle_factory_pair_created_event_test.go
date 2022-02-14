package exchange

import (
	"testing"
)

func TestHandleFactoryPairCreatedEvent(t *testing.T) {
	//testCase := &TestCase{}
	//createPairEventYaml, err := ioutil.ReadFile("./testdata/TestCreatePairEvent.yaml")
	//require.NoError(t, err)
	//
	//err = yaml.Unmarshal(createPairEventYaml, &testCase)
	//require.NoError(t, err)
	//
	//intrinsics := NewTestIntrinsics(testCase)
	//sg := NewTestSubgraph(intrinsics)
	//
	//err = sg.Init()
	//require.NoError(t, err)
	//
	//for _, ev := range testCase.Events {
	//	err := sg.HandleEvent(ev.Event)
	//	if err != nil {
	//		t.Errorf(err.Error())
	//	}
	//}
	//
	//bundle := intrinsics.store["bundle"]["1"].(*Bundle)
	//require.NotNil(t, bundle)
	//
	//factory := intrinsics.store["pancake_factory"][FactoryAddress].(*PancakeFactory)
	//require.NotNil(t, factory)
	//require.Equal(t, "1", factory.TotalPairs.String())
	//
	//pair := intrinsics.store["pair"]["0x00"].(*Pair)
	//require.NotNil(t, pair)
	//require.Equal(t, "0x00", pair.Token0)
	//require.Equal(t, "0x01", pair.Token1)
	//require.Equal(t, "token.0.symbol-token.1.symbol", pair.Name)
	//
	//token0 := intrinsics.store["token"]["0x00"].(*Token)
	//require.NotNil(t, token0)
	//require.Equal(t, "0", token0.Decimals.String())
	//require.Equal(t, "token.0.name", token0.Name)
	//require.Equal(t, "token.0.symbol", token0.Symbol)
	//
	//token1 := intrinsics.store["token"]["0x01"].(*Token)
	//require.NotNil(t, token1)
	//require.Equal(t, "10", token1.Decimals.String())
	//require.Equal(t, "token.1.name", token1.Name)
	//require.Equal(t, "token.1.symbol", token1.Symbol)
	//
	//require.True(t, sg.IsDynamicDataSource("0x00"))
	//
	//require.Equal(t, "0x00", sg.getPairAddressForTokens("0x00", "0x01"))
}
