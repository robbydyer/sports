import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Image from 'react-bootstrap/Image';
import stocksstocks from './stock.png';
import Form from 'react-bootstrap/Form';
import { GetStatus, CallMatrix } from './util';

class ImageBoard extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            "enabled": false,
            "memcache": false,
            "diskcache": false
        };
    }
    async componentDidMount() {
        await GetStatus("stocks/status", (val) => {
            this.setState({ "enabled": val })
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

    render() {
        return (
            <Container fluid>
                <Row className="text-center"><Col><Image src={stocksstocks} style={{ height: '100px', width: 'auto' }} fluid /></Col></Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="stocksenabler" label="Enable/Disable" checked={this.state.enabled}
                            onChange={() => this.handleSwitch("stocks/enable", "stocks/disable", "enabled")} />
                    </Col>
                </Row>
            </Container>
        )
    }
}

export default ImageBoard;