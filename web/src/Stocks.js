import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Image from 'react-bootstrap/Image';
import stocksstocks from './stock.png';
import Form from 'react-bootstrap/Form';
import { GetStatus, CallMatrix, MatrixPost } from './util';

class Stocks extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            "enabled": false,
            "scroll": false,
        };
    }
    async componentDidMount() {
        this.updateStatus()
    }
    async updateStatus() {
        await GetStatus("stocks/status", (val) => {
            this.setState({ "enabled": val })
        })
        await GetStatus(`stocks/scrollstatus`, (val) => {
            this.setState({
                "scroll": val,
            })
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
    handleJump = (board) => {
        MatrixPost("jump", `{"board":"${board}"}`)
        this.updateStatus()
    }

    render() {
        return (
            <Container fluid>
                <Row className="text-center"><Col><Image src={stocksstocks} style={{ height: '100px', width: 'auto' }} onClick={() => this.handleJump("stocks")} fluid /></Col></Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="stocksenabler" label="Enable/Disable" checked={this.state.enabled}
                            onChange={() => this.handleSwitch("stocks/enable", "stocks/disable", "enabled")} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="stocksscroller" label="Scroll Mode" checked={this.state.scroll}
                            onChange={() => this.handleSwitch(`stocks/scrollon`, `stocks/scrolloff`, "scroll")} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Button variant="primary" onClick={() => this.handleJump("stocks")}>Jump</Button>
                    </Col>
                </Row>
            </Container>
        )
    }
}

export default Stocks;