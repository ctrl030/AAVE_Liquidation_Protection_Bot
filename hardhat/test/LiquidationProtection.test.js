const { expect } = require("chai");
const path = require('path');
const fs = require('fs');
const bent = require('bent')
const getJSON = bent('json')
const convertHex = require('convert-hex')

const WETH = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2";
const LENDING_POOL = "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9";
const DAI = "0x6B175474E89094C44Da98b954EedeAC495271d0F";
const AETH = "0x030bA81f1c18d280636F32af80b9AAd02Cf0854e";

const MAX_UINT_AMOUNT =
  '115792089237316195423570985008687907853269984665640564039457584007913129639935';

describe('LiquidationProtection', () => {
  let lendingPool;
  let wETH;
  let aToken;

  let owner;
  let user;
  let addrs;

  let protection;
  let testHelper;

  before(async () => {
    let accountsPromise = ethers.getSigners();

    lendingPool = readABI('LendingPool.json', LENDING_POOL);
    wETH = readABI('WETH9.json', WETH);
    aToken = readABI('AToken.json', AETH);

    [owner, user, ...addrs] = await accountsPromise;

    let LiquidationProtection = await ethers.getContractFactory("LiquidationProtection");
    protection = await LiquidationProtection.deploy();

    let TestHelper = await ethers.getContractFactory("TestHelper");
    testHelper = await TestHelper.deploy();
  });

  it('deploys a contract', async () => {
    expect(protection).to.be.ok;
    expect(testHelper).to.be.ok;
  });

  let amount = web3.utils.toWei('1', 'ether');

  it('wraps eth', async () => {
    let result = await wETH.methods.deposit().send({
      from: user.address, value: amount,
    });
    expect(result.events.Deposit).to.be.ok;

    await testHelper.connect(user).assertWEth(amount);
  });

  it('deposits into LendingPool', async() => {
    let result = await wETH.methods.approve(LENDING_POOL, amount).send({
      from: user.address,
    });
    expect(result.events.Approval).to.be.ok;

    result = await lendingPool.methods.deposit(WETH, amount, user.address, 0).send({
      from: user.address,
    });
    expect(result.events.Deposit).to.be.ok;

    await testHelper.connect(user).assertAEth(amount);
  });

  it ('cannot borrow too much DAI', async() => {
    let borrowAmount = "4300000000000000000000";
    let gotError;
    try {
      await lendingPool.methods.borrow(DAI, borrowAmount, 1, 0, user.address).send({
         from: user.address,
      });
      throw null;
    } catch (error) {
      gotError = error;
    }
    expect(gotError).to.be.ok;
  });

  it('borrows DAI', async() => {
    let borrowAmount = "801000000000000000000";
    let result = await lendingPool.methods.borrow(DAI, borrowAmount, 1, 0, user.address).send({
       from: user.address,
    });
    expect(result.status).to.be.ok;

    await testHelper.connect(user).assertDAI(borrowAmount);
  });

  it('registers for protection', async() => {
    // Registration proceeds in two steps.
    // 1. The user first approves the protection contract to withdraw the collateral.
    let result = await aToken.methods.approve(protection.address, MAX_UINT_AMOUNT).send({
      from: user.address,
    });
    expect(result.events.Approval).to.be.ok;

    // 2. The user then registers with the protection contract.
    result = await (await protection.connect(user).register(WETH, DAI, 0)).wait();
    expect(result.events[0].event).to.equal('ProtectionRegistered');
  });

  let oneInchSwapCalldata;

  it ('obtains calldata', async() => {
    let result = await getJSON(`https://api.1inch.exchange/v2.0/swap?fromTokenAddress=${WETH}&toTokenAddress=${DAI}&amount=${amount}&fromAddress=${protection.address}&slippage=0.5&disableEstimate=true`);
    oneInchSwapCalldata = result.tx.data;
    expect(oneInchSwapCalldata).to.be.ok;
  });

  /**
   *  // TODO(greatfilter): the swap here is flaky. It should be debugged.
  it ('can 1inch swap', async() => {
    let result = await wETH.methods.deposit().send({
      from: owner.address, value: amount,
    });
    expect(result.events.Deposit).to.be.ok;

    result = await wETH.methods.transfer(protection.address, amount).send({
      from: owner.address,
    });
    expect(result.events.Transfer).to.be.ok;

    result = await (await protection.oneInchSwap(WETH,
        convertHex.hexToBytes(oneInchSwapCalldata))).wait();
    expect(result.status).to.be.ok;
  }); */

  it ('executes', async() => {
    let result = await (await protection.connect(owner).execute(
        user.address, convertHex.hexToBytes(oneInchSwapCalldata))).wait();
    expect(result.status).to.be.ok;
  });
});

function readABI(name, address) {
  p = path.resolve(__dirname, "..", "abis", name);
  return new web3.eth.Contract(JSON.parse(fs.readFileSync(p, 'utf8')), address);
}
