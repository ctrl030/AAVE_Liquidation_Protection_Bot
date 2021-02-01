import React from 'react';
import ReactDOM from 'react-dom';
import detectEthereumProvider from '@metamask/detect-provider';

const regeneratorRuntime = require("regenerator-runtime");

let provider;

(async () => {
  provider = await detectEthereumProvider();
  if (!provider) {
    console.log('Please install MetaMask!');
  }
})();

function isHexAddress(str) {
  return /^0x[a-fA-F0-9]{40}$/i.test(str)
}

const emptyValues = {
  'collateral-name': '',
  'collateral-address': '',
  'collateral-amount': '',
  'debt-name': '',
  'debt-address': '',
  'debt-amount': '',
  'current-ratio': '',
  'liquidation-threshold': '',
};

function addressLink(addr) {
  if (addr.length < 0) {
    return <td />;
  }
  return (<td>
     <a href={'https://etherscan.io/address/'.concat(addr)}>{addr}</a>
  </td>);
}

class ProtectionWidget extends React.Component {
  constructor(props) {
    super(props);

    this.state = emptyValues;
  }

  render() {
    return (
      <table>
      <tr>
        <td>Wallet Address</td>
        <td />
        <td><div id='connect-button'>
          <button onClick={this.connectClicked}>Connect to Metamask</button>
        </div></td>
      </tr>
      <tr>
        <td>Collateral</td>
        <td><b>{this.state['collateral-name']}</b></td>
        <td>{this.state['collateral-amount']}</td>
      </tr>
      <tr>
        <td></td>
        <td></td>
        {addressLink(this.state['collateral-address'])}
      </tr>
      <tr>
        <td>Debt</td>
        <td><b>{this.state['debt-name']}</b></td>
        <td>{this.state['debt-amount']}</td>
      </tr>
      <tr>
        <td></td>
        <td></td>
        {addressLink(this.state['debt-address'])}
      </tr>
      <tr>
        <td>Current Ratio</td>
        <td></td>
        <td>{this.state['current-ratio']}</td>
      </tr>
      <tr>
        <td>Liquidation Threshold</td>
        <td></td>
        <td>{this.state['liquidation-threshold']}</td>
      </tr>
      </table>
    );
  }

  connectClicked = (e) => {
    provider.request({ method: 'eth_requestAccounts' })
      .then(accounts => {
        let account = accounts[0];
        document.getElementById('connect-button').innerHTML = account;
        return fetch('http://localhost:3000/api/state?address='.concat(account));
      }).then(response => response.json())
      .then(json => {
        this.setState(json);
      });
  }
}

class App extends React.Component {
  constructor(props) {
    super(props);
  }

  render() {
    return <ProtectionWidget />;
  }
}

// Take this component's generated HTML and put it on the page (in the DOM).
ReactDOM.render(<App />, document.getElementById('app'));
