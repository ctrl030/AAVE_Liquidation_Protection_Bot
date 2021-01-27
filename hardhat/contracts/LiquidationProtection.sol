// SPDX-License-Identifier: agpl-3.0.
pragma solidity 0.6.12;
pragma experimental ABIEncoderV2;

import "hardhat/console.sol";

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";

import "../flashloan/interfaces/IFlashLoanReceiver.sol";
import "../interfaces/ILendingPool.sol";
import "../interfaces/ILendingPoolAddressesProvider.sol";
import "../protocol/libraries/types/DataTypes.sol";

contract LiquidationProtection is IFlashLoanReceiver {
  address payable constant WETH9 = payable(0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2);
  address constant ADDRESSES_PROVIDER_ADDRESS = 0xB53C1a33016B2DC2fF3653530bfF1848a515c8c5;
  address constant LENDING_POOL_ADDRESS = 0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9;
  address constant ONE_INCH = 0x111111125434b319222CdBf8C261674aDB56F3ae;

  address payable immutable public BOT;
  ILendingPoolAddressesProvider immutable public override ADDRESSES_PROVIDER;
  ILendingPool immutable public override LENDING_POOL;

  event ProtectionRegistered(
    address user,
    address aToken,
    address debtAsset,
    uint16 triggerRatio,
    uint gasValue
  );

  event ProtectionExercised(
    address user,
    uint repaid,
    uint redeemed,
    uint swappedTo
  );

  struct Protection {
    address aToken;
    address debtAsset;
    uint16 triggerRatio;
  }

  struct FlashParams {
    address user;
    address aToken;
    address debtAsset;
    uint sAmount;
    uint vAmount;
  }

  // TODO(greatfilter): this does not allow setting multiple protections per user.
  mapping(address => Protection) public protections;

  constructor() public {
    BOT = msg.sender;
    ADDRESSES_PROVIDER = ILendingPoolAddressesProvider(ADDRESSES_PROVIDER_ADDRESS);
    LENDING_POOL = ILendingPool(LENDING_POOL_ADDRESS);
  }

  /**
   * @param _collateral collateral asset address
   * @param _debt debt asset address
   * @param _triggerRatio LTV to trigger (in units of 1 / 256)
   */
  function register(address _collateral, address _debt, uint16 _triggerRatio) public payable {
    verifyConfiguration(_collateral, _debt);

    address aToken = toAToken(_collateral);
    verifyApproval(aToken);

    BOT.transfer(msg.value);
    // TODO(greatfilter): is there an API to check the existing ratios?
    protections[msg.sender] = Protection(aToken, _debt, _triggerRatio);

    emit ProtectionRegistered(msg.sender, aToken, _debt, _triggerRatio, msg.value);
  }

  /**
   * @dev Called by the bot to execute the debt repayment.
   * @param _user the owning account
   * @param _oneInchSwapCalldata specifies a swap from the underyling asset of the collateral to the
   *   debt asset with the BOT as the owner of the assets and an amount of collateral matching the
   *   _amount parameter.
   */
  function execute(address _user, bytes memory _oneInchSwapCalldata) public restricted {
    Protection memory protection = protections[_user];

    // TODO(greatfilter): add a chainlink lookup here to verify the ratio.
    (uint sAmount, uint vAmount) = getDebtAmount(protection.debtAsset, _user);
    require(sAmount + vAmount > 0, "debt not found");

    FlashParams memory fp = FlashParams(
        _user, protection.aToken, protection.debtAsset, sAmount, vAmount);
    executeStackHelper(protection.debtAsset, sAmount + vAmount,
        abi.encode(fp, _oneInchSwapCalldata));

    delete protections[_user];
  }

  /**
   * @dev called by execute to avoid "Stack too deep" errors.
   */
  function executeStackHelper(address _debtAsset, uint amount, bytes memory params) private {
    address[] memory assets = new address[](1);
    assets[0] = _debtAsset;
    uint[] memory amounts = new uint[](1);
    amounts[0] = amount;
    uint[] memory modes = new uint[](1);
    modes[0] = 0;
    LENDING_POOL.flashLoan(address(this), assets, amounts, modes, address(this), params, 0);
  }

  // Implements the flashloan callback.
  function executeOperation(
      address[] calldata /* _assets */,
      uint[] calldata _amounts,
      uint[] calldata _premiums,
      address /* _initiator */,
      bytes calldata _params) external override returns (bool) {
    (FlashParams memory fp, bytes memory oneInchSwapCalldata)
        = abi.decode(_params, (FlashParams, bytes));
    require(IERC20(fp.debtAsset).approve(0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9,
        type(uint).max), 'failed to approve the lending pool');
    uint repaid = 0;
    if (fp.sAmount > 0) {
      repaid += LENDING_POOL.repay(fp.debtAsset, fp.sAmount, 1, fp.user);
    }
    if (fp.vAmount > 0) {
      repaid += LENDING_POOL.repay(fp.debtAsset, fp.vAmount, 2, fp.user);
    }

    uint redeemed = retrieveCollateral(fp.user, fp.aToken);

    // Swaps collateral to debt using 1inch.
    uint amount = oneInchSwap(fp.aToken, oneInchSwapCalldata);

    distributeProceeds(fp, _amounts[0] + _premiums[0]);

    emit ProtectionExercised(fp.user, repaid, redeemed, amount);
    return true;
  }

  function oneInchSwap(address _aToken, bytes memory _oneInchSwapCalldata) public returns (uint) {
    address asset = underlyingAsset(_aToken);
    // Grants 1inch approval to make the swap.
    IERC20(asset).approve(ONE_INCH, type(uint).max);
    (bool s, bytes memory v) = ONE_INCH.call(_oneInchSwapCalldata);  // Performs the swap.
    require(s, "one inch swap failed");
    return abi.decode(v, (uint));
  }

  function retrieveCollateral(address _user, address _aToken) private returns (uint) {
    uint collateralAmount = IERC20(_aToken).balanceOf(_user);
    require(
        IERC20(_aToken).transferFrom(_user, address(this), collateralAmount),
        'collateral transfer failed');
    address asset = underlyingAsset(_aToken);
    // Withdraws aTokens to underlying asset so it can be used to repay the loan.
    return LENDING_POOL.withdraw(asset, type(uint).max, address(this));
  }

  function underlyingAsset(address _aToken) public returns (address) {
    (bool s, bytes memory v) = _aToken.call(abi.encodeWithSignature("UNDERLYING_ASSET_ADDRESS()"));
    require(s, "failed to get underlying asset");
    return abi.decode(v, (address));
  }

  function distributeProceeds(FlashParams memory _fp, uint _flashLoanAmount) private {
    IERC20 asset = IERC20(_fp.debtAsset);
    // Approves the lending pool to take payment.
    require(asset.approve(LENDING_POOL_ADDRESS, _flashLoanAmount),
        'failed to approve flash loan repayment');
    uint remaining = asset.balanceOf(address(this)) - _flashLoanAmount;
    // Transfers the remainder back to the user.
    require(asset.transfer(_fp.user, remaining), "transferring remainder to user failed");
  }

  function getDebtAmount(address _asset, address _user) private view returns (uint, uint) {
    (IERC20 stableDebtToken, IERC20 variableDebtToken) = toDebtTokens(_asset);
    return (stableDebtToken.balanceOf(_user), variableDebtToken.balanceOf(_user));
  }

  function verifyConfiguration(address _collateral, address _debt) private view {
    // TODO(greatfilter): consider moving these checks off chain to save gas.

    // Determines the indices of the assets.
    address[] memory reserves = LENDING_POOL.getReservesList();
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
    uint mask = LENDING_POOL.getUserConfiguration(msg.sender).data;
    uint collateralMask = 2 ** (2 * uint(collateralIndex) + 1);
    require(mask & collateralMask == collateralMask, "wrong collateral configuration");
    uint debtMask = 2 ** (2 * uint(debtIndex));
    require(mask & debtMask == debtMask, "wrong debt configuration");
  }

  function verifyApproval(address _aToken) private view {
    require(IERC20(_aToken).allowance(msg.sender, address(this)) == type(uint).max,
            "incorrect allowance value");
  }

  function toAToken(address _asset) private view returns (address) {
    DataTypes.ReserveData memory data = LENDING_POOL.getReserveData(_asset);
    return data.aTokenAddress;
  }

  function toDebtTokens(address _asset) private view returns (IERC20, IERC20) {
    DataTypes.ReserveData memory data = LENDING_POOL.getReserveData(_asset);
    return (IERC20(data.stableDebtTokenAddress), IERC20(data.variableDebtTokenAddress));
  }

  modifier restricted() {
    require(msg.sender == BOT);
    _;
  }
}
