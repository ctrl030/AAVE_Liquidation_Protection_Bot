var web3 = new Web3(Web3.givenProvider);

var instance;
var testingUser;
var botContractAddress = "";

$(document).ready(async function () {
  var accounts = await window.ethereum.enable();

  instance = new web3.eth.Contract(abi, botContractAddress, {
    from: accounts[0],
  });

  user = accounts[0];

});
