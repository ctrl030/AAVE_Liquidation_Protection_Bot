// SPDX-License-Identifier: agpl-3.0.
pragma solidity 0.6.12;
pragma experimental ABIEncoderV2;

// import "hardhat/console.sol";

import "../node_modules/@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "../flashloan/interfaces/IFlashLoanReceiver.sol";
import "../interfaces/ILendingPool.sol";
import "../interfaces/ILendingPoolAddressesProvider.sol";
import "../protocol/libraries/types/DataTypes.sol";

contract RepaymentExecutor is IFlashLoanReceiver {
  address constant ADDRESSES_PROVIDER_ADDRESS = 0xB53C1a33016B2DC2fF3653530bfF1848a515c8c5;
  address constant LENDING_POOL_ADDRESS = 0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9;
  address constant ONE_INCH = 0x111111125434b319222CdBf8C261674aDB56F3ae;

  ILendingPoolAddressesProvider immutable public override ADDRESSES_PROVIDER;
  ILendingPool immutable public override LENDING_POOL;

  struct FlashParams {
    address user;
    address aToken;
    address debtAsset;
    uint sAmount;
    uint vAmount;
  }

  bytes32 constant EIP712DOMAIN_TYPEHASH = keccak256(
      "EIP712Domain(string name,string version,uint256 chainId,string salt)"
  );
  bytes32 constant DELEGATE_TYPEHASH = keccak256(
      "Delegate(address delegate)"
  );

  bytes32 immutable DOMAIN_SEPARATOR;

  struct EIP712Domain {
    string  name;
    string  version;
    uint256 chainId;
    string salt;
  }

  struct Delegate {
    address delegate;
  }

  constructor() public {
    ADDRESSES_PROVIDER = ILendingPoolAddressesProvider(ADDRESSES_PROVIDER_ADDRESS);
    LENDING_POOL = ILendingPool(LENDING_POOL_ADDRESS);

    uint256 cId;
    assembly {
      cId := chainid()
    }
    DOMAIN_SEPARATOR = hash(EIP712Domain({
        name: "AAVE Liquidation Protection Bot",
        version: '1',
        chainId: cId,
        salt: "SU%N6gmumvj.A{@B,SdWXtVgg(Bof9SA"
    }));
  }

  /**
   * @dev Repays a loan using a flash loan, then repays the flash loan by redeeming the collateral
   *   and converting it to the loan asset type using 1inch.
   * @param _user the account owner
   * @param _signature signature of the certificate
   * @param _cAsset the underlying collateral asset
   * @param _dAsset the underyling debt asset
   * @param _oneInchSwapCalldata specifies a swap from the underyling asset of the collateral to the
   *   debt asset with the sender as the owner of the assets and an amount of collateral matching
   *   the _amount parameter.
   */
  function execute(address _user, bytes memory _signature, address _cAsset, address _dAsset,
      bytes memory _oneInchSwapCalldata) public {
    verifySignature(_user, _signature);

    (uint sAmount, uint vAmount) = getDebtAmount(_dAsset, _user);
    require(sAmount + vAmount > 0, "debt not found");
    uint amount = sAmount + vAmount;
    address aToken = toAToken(_cAsset);
    FlashParams memory fp = FlashParams(_user, aToken, _dAsset, sAmount, vAmount);
    bytes memory params = abi.encode(fp, _oneInchSwapCalldata);

    address[] memory assets = new address[](1);
    assets[0] = _dAsset;
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
    require(IERC20(fp.debtAsset).approve(LENDING_POOL_ADDRESS, type(uint).max),
        'failed to approve the lending pool');
    uint repaid = 0;
    if (fp.sAmount > 0) {
      repaid += LENDING_POOL.repay(fp.debtAsset, fp.sAmount, 1, fp.user);
    }
    if (fp.vAmount > 0) {
      repaid += LENDING_POOL.repay(fp.debtAsset, fp.vAmount, 2, fp.user);
    }

    /* uint redeemed = */ redeemCollateral(fp.user, fp.aToken);

    // Swaps collateral to debt using 1inch.
    /* uint amount = */ oneInchSwap(fp.aToken, oneInchSwapCalldata);

    distributeProceeds(fp, _amounts[0] + _premiums[0]);

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

  function redeemCollateral(address _user, address _aToken) private returns (uint) {
    uint collateralAmount = IERC20(_aToken).balanceOf(_user);
    require(IERC20(_aToken).transferFrom(_user, address(this), collateralAmount),
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

  function verifySignature(address _user, bytes memory _signature) private view {
    bytes32 digest = keccak256(abi.encodePacked(
        "\x19\x01",
        DOMAIN_SEPARATOR,
        hash(Delegate({ delegate: msg.sender }))
    ));
    require(recoverSigner(digest, _signature) == _user, "signer did not match");
  }

  function recoverSigner(bytes32 digest, bytes memory _signature) private pure returns (address) {
    require(_signature.length == 65, "wrong signature length");

    // Divides the signature in r, s and v variables
    bytes32 r;
    bytes32 s;
    uint8 v;
    assembly {
      r := mload(add(_signature, 32))
      s := mload(add(_signature, 64))
      v := byte(0, mload(add(_signature, 96)))
    }

    // Version of signature should be 27 or 28, but 0 and 1 are also possible versions
    if (v < 27) {
      v += 27;
    }
    require(v == 27 || v == 28, "v was not 27 or 28");

    return ecrecover(digest, v, r, s);
  }

  function hash(EIP712Domain memory eip712Domain) private pure returns (bytes32) {
    return keccak256(abi.encode(
        EIP712DOMAIN_TYPEHASH,
        keccak256(bytes(eip712Domain.name)),
        keccak256(bytes(eip712Domain.version)),
        eip712Domain.chainId,
        keccak256(bytes(eip712Domain.salt))
    ));
  }

  function hash(Delegate memory delegate) private pure returns (bytes32) {
    return keccak256(abi.encode(
        DELEGATE_TYPEHASH,
        delegate.delegate
    ));
  }

  function getDebtAmount(address _asset, address _user) private view returns (uint, uint) {
    (IERC20 stableDebtToken, IERC20 variableDebtToken) = toDebtTokens(_asset);
    return (stableDebtToken.balanceOf(_user), variableDebtToken.balanceOf(_user));
  }

  function toAToken(address _asset) private view returns (address) {
    DataTypes.ReserveData memory data = LENDING_POOL.getReserveData(_asset);
    return data.aTokenAddress;
  }

  function toDebtTokens(address _asset) private view returns (IERC20, IERC20) {
    DataTypes.ReserveData memory data = LENDING_POOL.getReserveData(_asset);
    return (IERC20(data.stableDebtTokenAddress), IERC20(data.variableDebtTokenAddress));
  }
}
