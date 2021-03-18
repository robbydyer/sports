import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Form from 'react-bootstrap/Form';
import Image from 'react-bootstrap/Image';
import pgalogo from './pga.png'
import { GetStatus, CallMatrix } from './util';

class Pga extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            "stats": false,
        };
    }
    async componentDidMount() {
        await GetStatus(`pga/stats/status`, (val) => {
            this.setState({
                "stats": val,
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

    logosrc() {
        return pgalogo;
    }
    render() {
        return (
            <Container fluid>
                <Row className="text-center"><Col><Image src={this.logosrc()} style={{ height: '100px', width: 'auto' }} fluid /></Col></Row>
                <Row className="text-center">
                    <Col>
                        <Form.Switch id="stats" label="Enable/Disable" checked={this.state.stats}
                            onChange={() => this.handleSwitch(`pga/stats/enable`, `pga/stats/disable`, "stats")} />
                    </Col>
                </Row>
            </Container>
        )
    }
}

export default Pga;