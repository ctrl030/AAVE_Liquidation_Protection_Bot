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

const etherscanUri = 'https://etherscan.io/address/';

class AddressInput extends React.Component {
  constructor(props) {
    super(props);

    this.onValueChange = props.onValueChange;
    this.state = { address: '' };
  }

  render() {
    return (<div>
      <input value={this.state.address}
             onChange={ev => this.onInputChange(ev.target.value)}
             size="42" />
      <div id='address-status'>Please enter a wallet address.</div>
    </div>);
  }

  onInputChange(address) {
    this.setState({address: address});
    if (isHexAddress(address)) {
      document.getElementById('address-status').innerHTML = '';
      fetch('http://localhost:3000/api/state?address='.concat(address))
          .then(response => response.json())
          .then(data => this.onValueChange(data));
    } else {
      document.getElementById('address-status').innerHTML = '<i>Please enter a wallet address.</i>';
      this.onValueChange(emptyValues);
    }
  }
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
        <td><AddressInput onValueChange={data => {
          this.setState(data);
        }} /></td>
      </tr>
      <tr>
        <td>Collateral</td>
        <td><b>{this.state['collateral-name']}</b></td>
        <td>{this.state['collateral-amount']}</td>
      </tr>
      <tr>
        <td></td>
        <td></td>
        {this.addressLink(this.state['collateral-address'])}
      </tr>
      <tr>
        <td>Debt</td>
        <td><b>{this.state['debt-name']}</b></td>
        <td>{this.state['debt-amount']}</td>
      </tr>
      <tr>
        <td></td>
        <td></td>
        {this.addressLink(this.state['debt-address'])}
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

  addressLink(addr) {
    if (addr.length < 0) {
      return <td />;
    }
    return (<td>
       <a href={etherscanUri.concat(addr)}>{addr}</a>
    </td>);
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
