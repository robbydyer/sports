import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Image from 'react-bootstrap/Image';
import server from './server.png';
import Form from 'react-bootstrap/Form';
import { GetStatus, CallMatrix, MatrixPost } from './util';

class Sys extends React.Component {
    constructor(props) {
        super(props);
        this.state = { "enabled": false };
    }
    async componentDidMount() {
        this.updateStatus()
    }

    async updateStatus() {
        await GetStatus(`sys/status`, (val) => {
            this.setState({
                "enabled": val,
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
                <Row className="text-center"><Col><Image src={server} style={{ height: '100px', width: 'auto' }} onClick={() => this.handleJump("sys")} fluid /></Col></Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="sysenabler" label="Enable/Disable" checked={this.state.enabled}
                            onChange={() => this.handleSwitch(`sys/enable`, `sys/disable`, "enabled")} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Button variant="primary" onClick={() => this.handleJump("sys")}>Jump</Button>
                    </Col>
                </Row>
            </Container>
        )
    }
}

export default Sys;