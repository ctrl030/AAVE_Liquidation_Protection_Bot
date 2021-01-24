const { expect } = require("chai");
const path = require('path');
const fs = require('fs');

const WETH = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2";
const LENDING_POOL = "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9";
const DAI = "0x6B175474E89094C44Da98b954EedeAC495271d0F";

describe('LiquidationProtection', () => {
  let lendingPool;
  let wETH;
  let wETHGateway;

  let owner;
  let user;
  let addrs;

  before(async () => {
    let accountsPromise = ethers.getSigners();

    lendingPool = readABI('LendingPool.json', LENDING_POOL);
    wETH = readABI('WETH9.json', WETH);
    wETHGateway = readABI('WETHGateway.json', "0xDcD33426BA191383f1c9B431A342498fdac73488");

    [owner, user, ...addrs] = await accountsPromise;
  });

  let protection;
  let testHelper;

  beforeEach(async () => {
    let LiquidationProtection = await ethers.getContractFactory("LiquidationProtection");
    protection = await LiquidationProtection.deploy();

    let TestHelper = await ethers.getContractFactory("TestHelper");
    testHelper = await TestHelper.deploy();
  });

  it('deploys a contract', () => {
    expect(protection).to.be.ok;
    expect(testHelper).to.be.ok;
  });

  let amount = web3.utils.toWei('1', 'ether');

  it('wraps eth', async () => {
    let result = await wETH.methods.deposit().send({
      from: user.address, value: amount
    });
    await testHelper.connect(user).assertWEth(amount);
  });

  it ('approves LendingPool contract', async() => {
    let result = await wETH.methods.approve(LENDING_POOL, amount).send({
      from: user.address,
    });
    expect(result.events.Approval).to.be.ok;
  });

  it('deposits into LendingPool', async() => {
    await testHelper.assertPaused(false);

    let result = await lendingPool.methods.deposit(WETH, amount, user.address, 0).send({
      from: user.address,
    });

    expect(result.events.Deposit).to.be.ok;
    await testHelper.connect(user).assertAEth(amount);
  });

  it('borrows DAI', async() => {
    let borrowAmount = String(amount / 2);
    let result = await lendingPool.methods.borrow(DAI, borrowAmount, 1, 0, user.address).send({
       from: user.address,
    });
    expect(result.status).to.be.ok;
    await testHelper.connect(user).assertDAI(borrowAmount);
  });
});

function readABI(name, address) {
  p = path.resolve(__dirname, "..", "abis", name);
  return new web3.eth.Contract(JSON.parse(fs.readFileSync(p, 'utf8')), address);
}
