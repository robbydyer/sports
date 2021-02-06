import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Image from 'react-bootstrap/Image';
import server from './server.png';

const MATRIX = "http://matrix.local:8080"

class Sys extends React.Component {
    constructor(props) {
        super(props);
    }
    callmatrix(path) {
        console.log(`Calling matrix Sys Board /clock/${path}`)
        fetch(`${MATRIX}/sys/${path}`, {
            method: "GET",
            mode: "cors",
        });
    }
    render() {
        return (
            <Container fluid>
                <Row className="text-center"><Col><Image src={server} style={{ height: '100px', width: 'auto' }} fluid /></Col></Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("enable")}>Enable</Button></Col></Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("disable")}>Disable</Button></Col></Row>
            </Container>
        )
    }
}

export default Sys;