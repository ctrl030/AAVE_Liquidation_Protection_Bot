const { expect } = require("chai");
const path = require('path');
const fs = require('fs');

const WETH = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2";
const LENDING_POOL = "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9";
const DAI = "0x6B175474E89094C44Da98b954EedeAC495271d0F";
const AETH = "0x030bA81f1c18d280636F32af80b9AAd02Cf0854e";

const MAX_UINT_AMOUNT =
  '115792089237316195423570985008687907853269984665640564039457584007913129639935';

describe('LiquidationProtection', () => {
  let lendingPool;
  let wETH;
  let wETHGateway;
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
    wETHGateway = readABI('WETHGateway.json', "0xDcD33426BA191383f1c9B431A342498fdac73488");
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
    expect(result.status).to.be.ok;

    let result2 = await (await testHelper.connect(user).assertWEth(amount)).wait();
    expect(result2.status).to.be.ok;
  });

  it ('approves LendingPool contract', async() => {
    let result = await wETH.methods.approve(LENDING_POOL, amount).send({
      from: user.address,
    });
    expect(result.events.Approval).to.be.ok;
  });

  it('deposits into LendingPool', async() => {
    let result = await lendingPool.methods.deposit(WETH, amount, user.address, 0).send({
      from: user.address,
    });

    expect(result.events.Deposit).to.be.ok;

    result = await (await testHelper.connect(user).assertAEth(amount)).wait();
    expect(result.status).to.be.ok;
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
    let borrowAmount = "1001000000000000000000";
    let result = await lendingPool.methods.borrow(DAI, borrowAmount, 1, 0, user.address).send({
       from: user.address,
    });
    expect(result.status).to.be.ok;
    let result2 = await (await testHelper.connect(user).assertDAI(borrowAmount)).wait();
    expect(result2.status).to.be.ok;
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
});

function readABI(name, address) {
  p = path.resolve(__dirname, "..", "abis", name);
  return new web3.eth.Contract(JSON.parse(fs.readFileSync(p, 'utf8')), address);
}
