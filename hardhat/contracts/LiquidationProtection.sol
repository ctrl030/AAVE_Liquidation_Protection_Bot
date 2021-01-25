// SPDX-License-Identifier: UNLICENSED
pragma solidity 0.7.3;

import "hardhat/console.sol";

contract LiquidationProtection {
  address constant LENDING_POOL = 0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9;

  address immutable public BOT;

  mapping(address=>address) public toAToken;

  event ProtectionRegistered(
    address user,
    address collateral,
    address debt,
    uint16 triggerRatio
  );

  struct Protection {
    address collateral;
    address debt;
    uint16 triggerRatio;
  }

  // TODO(greatfilter): this does not allow setting multiple protections per address.
  mapping(address => Protection) public protections;

  constructor() {
    BOT = msg.sender;

    // TODO(greatfilter): fill out remaining pairs.
    toAToken[0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2] =
      0x030bA81f1c18d280636F32af80b9AAd02Cf0854e;  // WETH9 to AETH.
  }

  /**
   * @param _collateral collateral asset address
   * @param _debt debt asset address
   * @param _triggerRatio LTV to trigger (in units of 1 / 256)
   */
  function register(address _collateral, address _debt, uint16 _triggerRatio) public {
    verifyConfiguration(_collateral, _debt);
    verifyApproval(_collateral);

    // TODO(greatfilter): is there an API to check the existing ratios?
    protections[msg.sender] = Protection(_collateral, _debt, _triggerRatio);

    emit ProtectionRegistered(msg.sender, _collateral, _debt, _triggerRatio);
  }

  function execute() public restricted {
  }

  function verifyConfiguration(address _collateral, address _debt) private {
    // TODO(greatfilter): consider moving these checks off chain to save gas.

    // Determines the indices of the assets.
    (bool s1, bytes memory v1) = LENDING_POOL.call(abi.encodeWithSignature("getReservesList()"));
    require(s1, "error listing reserves");
    address[] memory reserves = abi.decode(v1, (address[]));
    uint reservesLength = reserves.length;
    require(reservesLength < uint(type(int16).max), "reserve list too long");
    int16 collateralIndex = -1;
    int16 debtIndex = -1;
    for (uint i = 0; i < reservesLength; i++) {
      if (collateralIndex < 0 && _collateral == reserves[i]) {
        collateralIndex = int16(i);
      }
      if (debtIndex < 0 && _debt == reserves[i]) {
        debtIndex = int16(i);
      }
    }
    require(collateralIndex >= 0, "collateral not found");
    require(debtIndex >= 0, "debt not found");
    require(collateralIndex != debtIndex, "collateral must differ from debt");

    // Ensures that collateral and debt addresses are correctly configured.
    (bool s2, bytes memory v2) = LENDING_POOL.call(
      abi.encodeWithSignature("getUserConfiguration(address)", msg.sender));
    require(s2, "error getting user configuration");
    uint mask = abi.decode(v2, (uint));
    uint collateralMask = 2 ** (2 * uint(collateralIndex) + 1);
    require(mask & collateralMask == collateralMask, "wrong collateral configuration");
    uint debtMask = 2 ** (2 * uint(debtIndex));
    require(mask & debtMask == debtMask, "wrong debt configuration");
  }

  function verifyApproval(address _collateral) private {
    address aToken = toAToken[_collateral];
    (bool s, bytes memory value) = aToken.call(
      abi.encodeWithSignature("allowance(address,address)", msg.sender, address(this)));
    require(s, "error querying allowance");
    uint amount = abi.decode(value, (uint));
    require(amount == type(uint).max, "incorrect allowance value");
  }

  modifier restricted() {
    require(msg.sender == BOT);
    _;
  }
}
