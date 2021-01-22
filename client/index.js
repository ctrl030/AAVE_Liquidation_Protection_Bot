$(document).ready(async function () {
  const web3 = new Web3(
    `https://mainnet.infura.io/v3/7bc949eb46ef4d6cbe9412d5bc07cac7`
  );
  const web3Socket = new Web3(
    `wss://mainnet.infura.io/ws/v3/7bc949eb46ef4d6cbe9412d5bc07cac7`
  );

  // "parent" Chainlink ETH-USD contract: "0x5f4ec3df9cbd43714fe2740f5e3616155c5b8419"

  // aggregator that emits ETH-USD events
  var instance_5446;
  var instance_5446Address = "0x00c7a37b03690fb9f41b5c5af8131735c7275446";

  var testingUser;

  var accounts = await window.ethereum.enable();

  var roundId;

  var rowCounter = 1;

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

  console.log("Subscribing to Chainlink ETH-USD price...");

  // subscribing to aggregator websocket instance
  instance_5446_websocket.events
    .AnswerUpdated()
    .on("data", async function (event) {
      // console.log("instance_5446_websocket event");
      console.log(event);

      const EthPrice = Math.round(event.returnValues[0] / 100000000);
      console.log("ETH-USD price is: ", EthPrice);

      // $("#chainlinkPriceforETH").html(`EthPrice is: ${EthPrice}`);
      // $("#priceFeedBox").html(`EthPrice is: ${EthPrice}`);

      roundId = event.returnValues[1];

      $("#tableBody").prepend(
        `
          <tr class="tableRow">
            <th scope="row" id="rowCounter_${rowCounter}">${rowCounter}</th>
            <th scope="row" id="roundID_${rowCounter}">${roundId}</th>
            <th scope="row" id="price_${rowCounter}">${EthPrice}</th>
            <th scope="row" id="collateralValueTotal_${rowCounter}"></th>
            <th scope="row" id="wouldLiquidateAt_${rowCounter}"></th>
            
          </tr>
        `
      );

      rowCounter += 1;
    });
});
