require("@nomiclabs/hardhat-waffle");
require("@nomiclabs/hardhat-web3");

/**
 * @type import('hardhat/config').HardhatUserConfig
 */
module.exports = {
  solidity: "0.6.12",
  networks: {
    hardhat: {
      forking: {
        url: "https://mainnet.infura.io/v3/92e15d2bbf9b41d286544a680c4b23d0",
      },
      chainId: 1337  // Hardhat defaults to 31337, which causes errors with forked mainnet.
    }
  }
};
