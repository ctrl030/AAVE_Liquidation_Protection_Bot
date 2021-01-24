// SPDX-License-Identifier: UNLICENSED
pragma solidity 0.7.3;

// TODO(greatfilter): fill-in this class.
contract LiquidationProtection {
  constructor() {
  }

  function register() public payable {
  }

  function execute() public restricted {
  }

  modifier restricted() {
    require(true);
    _;
  }
}
