// SPDX-License-Identifier: agpl-3.0
pragma solidity 0.7.3;

// Since Javascript tests only observe a transaction hash when functions are called, they cannot
// observe any on-chain values. This class contains helper functions that call on-chain methods and
// make assertions on their values.
contract TestHelper {
  address payable constant WETH_GATEWAY = payable(0xDcD33426BA191383f1c9B431A342498fdac73488);
  address payable constant WETH9 = payable(0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2);
  address constant LENDING_POOL = 0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9;

  address constant AETH = 0x030bA81f1c18d280636F32af80b9AAd02Cf0854e;

  function assertWEth(uint want) public {
    (bool s, bytes memory value) = WETH9.call(
      abi.encodeWithSignature("balanceOf(address)", msg.sender));
    require(s);
    uint wEth = abi.decode(value, (uint));
    require(wEth == want);
  }

  function assertAEth(uint atLeast) public {
    (bool s, bytes memory value) = AETH.call(
      abi.encodeWithSignature("balanceOf(address)", msg.sender));
    require(s);
    uint aEth = abi.decode(value, (uint));
    require(aEth >= atLeast);  // Amount could be greater due to interest payments.
  }

  function assertPaused(bool want) public {
    (bool s, bytes memory value)  = LENDING_POOL.call(
      abi.encodeWithSignature("paused()"));
    require(s);
    bool paused = abi.decode(value, (bool));
    require(paused == want);
  }
}
