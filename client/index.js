// @dev need to manually update fromBlock for now
require("dotenv").config();
const Web3 = require("web3");
var web3 = new Web3(`https://mainnet.infura.io/v3/${process.env.INFURA_KEY}`);

// some kind of "parent" contract
var instance_8419;
var instance_8419Address = "0x5f4ec3df9cbd43714fe2740f5e3616155c5b8419";

// aggregator that emits ETH-USD events
var instance_5446;
var instance_5446Address = "0x00c7a37b03690fb9f41b5c5af8131735c7275446";

var testingUser;

// var botContractAddress = "";

$(document).ready(async function () {
  var accounts = await window.ethereum.enable();

  // "parent" instance
  instance_8419 = new web3.eth.Contract(abi_8419, instance_8419Address, {
    from: accounts[0],
  });

  // aggregator instance
  instance_5446 = new web3.eth.Contract(abi_5446, instance_5446Address, {
    from: accounts[0],
  });

  testingUser = accounts[0];

  console.log("parent instance_8419");
  console.log(instance_8419);

  console.log("aggregator instance_5446");
  console.log(instance_5446);

  // "parent" instance, trying to subscribe (this contract does not emit events though)
  instance_8419.events.AnswerUpdated().on("data", async function (event) {
    console.log("instance_8419 event");
    console.log(event);
    const EthPrice_8419 = await instance_8419.methods.latestAnswer().call();
    console.log("EthPrice_8419");
    console.log(String(EthPrice_8419));
  });

  // aggregator instance trying to subscribe (but doesn't seem to work yet)
  instance_5446.events.AnswerUpdated().on("data", async function (event) {
    console.log("instance_5446 event");
    console.log(event);
    const EthPrice_5446 = await instance_5446.methods.latestAnswer().call();
    console.log("EthPrice_5446");
    console.log(String(EthPrice_5446));
  });

  // aggregator instance
  // querying for all price updates on ETH-USD since "fromBlock:"
  // watch out, must be updated manually at the moment
  z_5446_AnswerUpdated_Array = await instance_5446.getPastEvents(
    "AnswerUpdated",
    {
      fromBlock: 11700814,
      toBlock: "latest",
    }
  );

  // console logging instance z_5446 , the aggregator, gives back array of objects
  console.log("z_5446_AnswerUpdated_Array");
  console.log(String(z_5446_AnswerUpdated_Array));

  // iterating through the answer array, console logging for each object
  for (let index = 0; index < z_5446_AnswerUpdated_Array.length; index++) {
    const element = z_5446_AnswerUpdated_Array[index];
    console.log("z_5446_AnswerUpdated_Array position " + index);
    console.log(element);

    // returnValues can be used to look deeper into each object
    // here we also round the price so its easier to read in the console
    let price = Math.round(element.returnValues[0] / 100000000);
    console.log("z_5446_AnswerUpdated_Array price, position " + index);
    console.log(price);
  }

  // comes back empty, correct, does not emit events
  z_8419_AnswerUpdated_Array = await instance_8419.getPastEvents(
    "AnswerUpdated",
    {
      fromBlock: 11700814,
      toBlock: "latest",
    }
  );

  console.log("z_8419_AnswerUpdated_Array");
  console.log(String(z_8419_AnswerUpdated_Array));
});
