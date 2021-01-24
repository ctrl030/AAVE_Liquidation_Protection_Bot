const { expect } = require("chai");
const path = require('path');
const fs = require('fs');

const WETH = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2";
const LENDING_POOL = "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9";

describe('LiquidationProtection', () => {
  let lendingPool;
  let wETH;
  let wETHGateway;

  let owner;
  let addr1;
  let addr2;
  let addrs;

  before(async () => {
    let accountsPromise = ethers.getSigners();

    lendingPool = readABI('LendingPool.json', LENDING_POOL);
    wETH = readABI('WETH9.json', WETH);
    wETHGateway = readABI('WETHGateway.json', "0xDcD33426BA191383f1c9B431A342498fdac73488");

    [owner, addr1, addr2, ...addrs] = await accountsPromise;
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
      from: addr1.address, value: amount
    });
    await testHelper.connect(addr1).assertWEth(amount);
  });

  it ('approves LendingPool contract', async() => {
    let result = await wETH.methods.approve(LENDING_POOL, amount).send({
      from:  addr1.address,
    });
    expect(result.events.Approval).to.be.ok;
  });

  it('deposits into LendingPool', async() => {
    await testHelper.assertPaused(false);

    let result = await lendingPool.methods.deposit(WETH, amount, addr1.address, 0).send({
      from:  addr1.address,
    });

    expect(result.events.Deposit).to.be.ok;
    await testHelper.connect(addr1).assertAEth(amount);
  });

  it('takes a loan', async() => {
  });
});

function readABI(name, address) {
  p = path.resolve(__dirname, "..", "abis", name);
  return new web3.eth.Contract(JSON.parse(fs.readFileSync(p, 'utf8')), address);
}
