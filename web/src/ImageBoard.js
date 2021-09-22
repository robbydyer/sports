import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Image from 'react-bootstrap/Image';
import imgimg from './image.png';
import Form from 'react-bootstrap/Form';
import { GetStatus, CallMatrix, MatrixPost } from './util';

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
        await this.updateStatus()
    }

    updateStatus = async () => {
        await GetStatus("img/status", (val) => {
            this.setState({ "enabled": val })
        })
        await GetStatus("img/memcachestatus", (val) => {
            this.setState({ "memcache": val })
        })
        await GetStatus("img/diskcachestatus", (val) => {
            this.setState({ "diskcache": val })
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
    handleJump = async (board) => {
        await MatrixPost("jump", `{"board":"${board}"}`)
        await this.updateStatus()
    }

    render() {
        return (
            <Container fluid>
                <Row className="text-center"><Col><Image src={imgimg} style={{ height: '100px', width: 'auto' }} onClick={() => this.handleJump("img")} fluid /></Col></Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="imgenabler" label="Enable/Disable" checked={this.state.enabled}
                            onChange={() => this.handleSwitch("img/enable", "img/disable", "enabled")} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="imgmem" label="Enable Memory Cache" checked={this.state.memcache}
                            onChange={() => this.handleSwitch("img/enablememcache", "img/disablememcache", "memcache")} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Form.Switch id="imgdisk" label="Enable Disk Cache" checked={this.state.diskcache}
                            onChange={() => this.handleSwitch("img/enablediskcache", "img/disablediskcache", "diskcache")} />
                    </Col>
                </Row>
                <Row className="text-left">
                    <Col>
                        <Button variant="primary" onClick={() => this.handleJump("img")}>Jump</Button>
                    </Col>
                </Row>
            </Container>
        )
    }
}

export default ImageBoard;