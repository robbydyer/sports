import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Image from 'react-bootstrap/Image';
import clock from './clock.png';

var BACKEND = "http://" + window.location.host

class Clock extends React.Component {
    constructor(props) {
        super(props);
    }
    callmatrix(path) {
        console.log(`Calling matrix Image Board /clock/${path}`)
        fetch(`${BACKEND}/api/clock/${path}`, {
            method: "GET",
            mode: "cors",
        });
    }
    render() {
        return (
            <Container fluid>
                <Row className="text-center"><Col><Image src={clock} style={{ height: '100px', width: 'auto' }} fluid /></Col></Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("enable")}>Enable</Button></Col></Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("disable")}>Disable</Button></Col></Row>
            </Container>
        )
    }
}

export default Clock;