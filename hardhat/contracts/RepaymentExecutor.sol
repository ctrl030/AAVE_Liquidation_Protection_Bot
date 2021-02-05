// SPDX-License-Identifier: agpl-3.0.
pragma solidity 0.6.12;
pragma experimental ABIEncoderV2;

import "../node_modules/@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "../flashloan/interfaces/IFlashLoanReceiver.sol";
import "../interfaces/ILendingPool.sol";
import "../interfaces/ILendingPoolAddressesProvider.sol";

contract RepaymentExecutor is IFlashLoanReceiver {
  address constant ADDRESSES_PROVIDER_ADDRESS = 0xB53C1a33016B2DC2fF3653530bfF1848a515c8c5;
  address constant LENDING_POOL_ADDRESS = 0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9;
  address constant ONE_INCH = 0x111111125434b319222CdBf8C261674aDB56F3ae;

  ILendingPoolAddressesProvider immutable public override ADDRESSES_PROVIDER;
  ILendingPool immutable public override LENDING_POOL;

  bytes32 constant EIP712DOMAIN_TYPEHASH = keccak256(
      "EIP712Domain(string name,string version,uint256 chainId,string salt)"
  );
  bytes32 constant DELEGATE_TYPEHASH = keccak256("Delegate(address delegate)");

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

  struct FlashParams {
    address user;  // Loan owner.
    uint sAmount;  // Stable debt amount.
    uint vAmount;  // Variable debt amount.
    // packed encodes (the AToken, its underlying asset, and the collateral amount)
    bytes packed;
    bytes oneInchCalldata;
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
   * @param _sDebtToken variable debt token
   * @param _vDebtToken stable debt token
   * @param _dAsset the underyling debt asset
   * @param _packed contains encoded parameters used only after the flash loan callback. See the
   *     FlashParams for a description of its contents.
   * @param _oneInchCalldata swap calldata obtained from the 1inch API
   */
  function execute(address _user, bytes memory _signature, address _sDebtToken, address _vDebtToken,
      address _dAsset, bytes memory _packed, bytes memory _oneInchCalldata) public {
    verifySignature(_user, _signature);

    uint debtAmount;
    bytes memory params;
    {
      uint vAmount = IERC20(_vDebtToken).balanceOf(_user);
      uint sAmount = IERC20(_sDebtToken).balanceOf(_user);
      require(sAmount + vAmount > 0, "debt not found");
      debtAmount = sAmount + vAmount;
      params = abi.encode(FlashParams(_user, sAmount, vAmount, _packed, _oneInchCalldata));
    }

    address[] memory assets = new address[](1);
    assets[0] = _dAsset;
    uint[] memory amounts = new uint[](1);
    amounts[0] = debtAmount;
    uint[] memory modes = new uint[](1);
    modes[0] = 0;
    LENDING_POOL.flashLoan(address(this), assets, amounts, modes, address(this), params, 0);
  }

  // Implements the flashloan callback.
  function executeOperation(
      address[] calldata _assets,
      uint[] calldata _amounts,
      uint[] calldata _premiums,
      address /* _initiator */,
      bytes calldata _params) external override returns (bool) {
    FlashParams memory fp = abi.decode(_params, (FlashParams));
    IERC20 debtAsset = IERC20(_assets[0]);
    uint flashLoanDebt = _amounts[0] + _premiums[0];
    require(debtAsset.approve(LENDING_POOL_ADDRESS, _amounts[0]),
        'failed to approve the lending pool');
    if (fp.sAmount > 0) {
      LENDING_POOL.repay(_assets[0], fp.sAmount, 1, fp.user);
    }
    if (fp.vAmount > 0) {
      LENDING_POOL.repay(_assets[0], fp.vAmount, 2, fp.user);
    }

    (address aToken, address cAsset, uint cAmount) = abi.decode(fp.packed,
        (address, address, uint));

    // Withdraws ATokens to the underlying asset.
    // Temporarily transfers ATokens into this contract.
    require(IERC20(aToken).transferFrom(fp.user, address(this), cAmount),
        'collateral transfer failed');
    // Withdraws the underyling asset (transforming the transferred ATokens).
    require(cAmount == LENDING_POOL.withdraw(cAsset, cAmount, address(this)),
        "withdrew less than the expected amount");

    // Swaps collateral to debt using 1inch.
    uint proceeds = oneInchSwap(cAsset, cAmount, fp.oneInchCalldata);

    // Distributes the proceeds.
    // Returns anything remaining back to the user.
    require(debtAsset.transfer(fp.user, proceeds - flashLoanDebt),
        "transferring remainder to user failed");
    // Approves the lending pool to take payment.
    require(debtAsset.approve(LENDING_POOL_ADDRESS, flashLoanDebt),
        'failed to approve flash loan repayment');

    return true;
  }

  function oneInchSwap(address _cAsset, uint _cAmount, bytes memory _oneInchSwapCalldata)
      private returns (uint) {
    IERC20(_cAsset).approve(ONE_INCH, _cAmount);  // Grants 1inch approval to make the swap.
    (bool s, bytes memory v) = ONE_INCH.call(_oneInchSwapCalldata);  // Performs the swap.
    require(s, "1inch swap failed");
    return abi.decode(v, (uint));
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

    // Version of the signature should be 27 or 28, but 0 and 1 are also possible.
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
    return keccak256(abi.encode(DELEGATE_TYPEHASH, delegate.delegate));
  }
}
