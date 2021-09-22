import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Form from 'react-bootstrap/Form';
import Image from 'react-bootstrap/Image';
import pgalogo from './pga.png'
import { GetStatus, CallMatrix, MatrixPost } from './util';

class Pga extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            "stats": false,
            "scroll": false,
        };
    }
    componentDidMount() {
        this.updateStatus()
    }
    updateStatus() {
        GetStatus(`pga/stats/status`, (val) => {
            this.setState({
                "stats": val,
            })
        })
        GetStatus(`pga/stats/scrollstatus`, (val) => {
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

    logosrc() {
        return pgalogo;
    }
    render() {
        return (
            <Container fluid>
                <Row className="text-center"><Col><Image src={this.logosrc()} style={{ height: '100px', width: 'auto' }} fluid /></Col></Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="pgastats" label="Enable/Disable" checked={this.state.stats}
                            onChange={() => this.handleSwitch(`pga/stats/enable`, `pga/stats/disable`, "stats")} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="pgascroll" label="Scroll Mode" checked={this.state.scroll}
                            onChange={() => this.handleSwitch(`pga/stats/scrollon`, `pga/stats/scrolloff`, "scroll")} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Button variant="primary" onClick={() => this.handleJump("pga")}>Jump</Button>
                    </Col>
                </Row>
            </Container>
        )
    }
}

export default Pga;