import React from 'react';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Form from 'react-bootstrap/Form';
import { GetStatus, CallMatrix } from './util.js';
import 'bootstrap/dist/css/bootstrap.min.css';

class Home extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            "screen": false,
            "webboard": false,
        };
    }
    async componentDidMount() {
        await GetStatus("status", (val) => {
            this.setState({ "screen": val })
        })
        await GetStatus("webboardstatus", (val) => {
            this.setState({ "webboard": val })
        })
    }

    handleSwitch = (apiOn, apiOff, stateVar) => {
        var currentState = this.state[stateVar]
        console.log("handle switch", currentState)
        if (currentState) {
            console.log("Turn off", apiOff)
            CallMatrix(apiOff);
        } else {
            console.log("Turn on", apiOn)
            CallMatrix(apiOn);
        }
        this.setState(prev => ({
            [stateVar]: !prev[stateVar],
        }))
    }

    disableAll = async () => {
        await CallMatrix("/disableall")
        this.props.doSync()
    }
    enableAll = async () => {
        await CallMatrix("/enableall")
        this.props.doSync()
    }

    render() {
        return (
            <Container fluid>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="screen" label="Screen On/Off" checked={this.state["screen"]} onChange={() => this.handleSwitch("screenon", "screenoff", "screen")} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="webboard" label="Web Board On/Off" checked={this.state["webboard"]} onChange={() => this.handleSwitch("webboardon", "webboardoff", "webboard")} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Button variant="primary" onClick={this.enableAll}>Enable All</Button>
                    </Col>
                    <Col>
                        <Button variant="primary" onClick={this.disableAll}>Disable All</Button>
                    </Col>
                </Row>
            </Container>
        );
    }
}
export default Home;