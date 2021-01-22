$(document).ready(async function () {
  const web3 = new Web3(
    `https://mainnet.infura.io/v3/7bc949eb46ef4d6cbe9412d5bc07cac7`
  );
  const web3Socket = new Web3(
    `wss://mainnet.infura.io/ws/v3/7bc949eb46ef4d6cbe9412d5bc07cac7`
  );

  /*
  // "parent" Chainlink contract "0x5f4ec3df9cbd43714fe2740f5e3616155c5b8419"
  */

  // aggregator that emits ETH-USD events
  var instance_5446;
  var instance_5446Address = "0x00c7a37b03690fb9f41b5c5af8131735c7275446";

  var testingUser;

  var accounts = await window.ethereum.enable();

  // aggregator instance
  instance_5446 = new web3.eth.Contract(abi_5446, instance_5446Address, {
    from: accounts[0],
  });

  // aggregator websocket instance
  instance_5446_websocket = new web3Socket.eth.Contract(
    abi_5446,
    instance_5446Address,
    {
      from: accounts[0],
    }
  );

  testingUser = accounts[0];

  // console.log("aggregator instance_5446");
  // console.log(instance_5446);
  /*
  // "parent" instance, trying to subscribe (this contract does not emit events though)
  instance_8419.events.AnswerUpdated().on("data", async function (event) {
    console.log("instance_8419 event");
    console.log(event);
    const EthPrice_8419 = await instance_8419.methods.latestAnswer().call();
    console.log("EthPrice_8419");
    console.log(String(EthPrice_8419));
  });
*/
  console.log("Subscribing to Chainlink ETH-USD price...");

  // subscribing to aggregator websocket instance
  instance_5446_websocket.events
    .AnswerUpdated()
    .on("data", async function (event) {
      // console.log("instance_5446_websocket event");
      console.log(event);

      const EthPrice = Math.round(event.returnValues[0] / 100000000);
      console.log("ETH-USD price is: ", EthPrice);

      $("#chainlinkPriceforETH").html(`EthPrice is: ${EthPrice}`);
      $("#priceFeedBox").html(`EthPrice is: ${EthPrice}`);
    });

  /*  
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
  */
});
