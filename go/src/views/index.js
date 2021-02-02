import 'regenerator-runtime/runtime'
import React from 'react';
import ReactDOM from 'react-dom';
import detectEthereumProvider from '@metamask/detect-provider';
import Web3 from 'web3';

const bent = require('bent')
const getJSON = bent('json')

const HOST = 'http://localhost:3000'
const MAX_UINT_AMOUNT =
  '115792089237316195423570985008687907853269984665640564039457584007913129639935';

let provider;
let web3;
let account;  // XXX: reduce the scope of this.

(async () => {
  provider = await detectEthereumProvider();
  if (!provider) {
    console.log('Please install MetaMask!');
    return;
  }
  web3 = new Web3(provider);
})();

function addressLink(addr) {
  if (addr.length < 0) {
    return <td />;
  }
  return (<td>
     <a href={'https://etherscan.io/address/'.concat(addr)}>{addr}</a>
  </td>);
}

class RegisterWidget extends React.Component {
  constructor(props) {
    super(props);

    this.state = { value: '' };
    this.registerCb = props.register;
  }

  componentDidUpdate(prevProps, prevState, snapshot) {
    this.checkEnabled();
  }

  onInputChange(value) {
    this.setState({value: value});
  }

  checkEnabled() {
    let threshold = parseFloat(this.props.threshold);
    let value = parseFloat(this.state.value);
    let button = document.getElementById('register-button');
    if (threshold > 0 && value > 0 && value < threshold) {
      button.removeAttribute('disabled');
      button.addEventListener('click', this.register);
    } else {
      button.disabled = true;
    }
  }

  register = () => {
    this.registerCb(this.state.value);
  }

  render() {
    return (<tr>
      <td>
        <button id='register-button' onClick={this.register} disabled>Register Protection</button>
      </td>
      <td>Custom Threshold</td>
      <td><input
          value={this.state.value}
          onChange={ev => this.setState({value: ev.target.value})} />
      </td>
    </tr>);
  }
}

class ProtectionWidget extends React.Component {
  constructor(props) {
    super(props);

    this.state = {
      'collateral-name': '',
      'collateral-address': '',
      'collateral-amount': '',
      'debt-name': '',
      'debt-address': '',
      'debt-amount': '',
      'current-ratio': '',
      'liquidation-threshold': '',
    };
  }

  render() {
    return (
      <table>
      <tr>
        <td>Wallet Address</td>
        <td />
        <td>
          <button id='connect-button' onClick={this.connectClicked}>Connect to Metamask</button>
        </td>
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
      <RegisterWidget threshold={this.state['liquidation-threshold']} register={this.register} />
      </table>
    );
  }

  connectClicked = async (e) => {
    let accounts = await provider.request({ method: 'eth_requestAccounts' })
    account = accounts[0];
    let cButton = document.getElementById('connect-button');
    cButton.innerHTML = account;
    cButton.disabled = true;
    let response = await fetch(HOST.concat('/api/state?address=').concat(account));
    let json = await response.json();
    this.setState(json);
  }

  register = async (value) => {
    console.log("register clicked with value =", value);
    let erc20ABI = await getJSON(HOST.concat('/api/abi?name=erc20'));
    let aToken = new web3.eth.Contract(erc20ABI, this.state['a-token-address']);
    let result = await aToken.methods.approve(
        this.state['protection-contract-address'], MAX_UINT_AMOUNT).send({
      from: account,
    });
    console.log('result=', result);

    let ratio = Math.floor(10000 * parseFloat(value));
    let protectionABI = await getJSON(HOST.concat('/api/abi?name=protection'));
    let protection = new web3.eth.Contract(
        protectionABI, this.state['protection-contract-address']);
    result = await protection.methods.register(this.state['collateral-address'],
        this.state['debt-address'], ratio).send({
      from: account,
      value: 9500000000,  // Gas used to execute flash repayment.
    });
    console.log('result=', result);
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
