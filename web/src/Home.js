import React from 'react';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import 'bootstrap/dist/css/bootstrap.min.css';

var BACKEND = "http://" + window.location.host

class Home extends React.Component {
    callmatrix(path) {
        console.log(`Calling matrix API ${path}`)
        fetch(`${BACKEND}/api/${path}`, {
            method: "GET",
            mode: "cors",
        });
    }
    render() {
        return (
            <Container fluid>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("screenon")}>Screen On </Button></Col></Row>
                <Row className="text-center"><Col><Button onClick={() => this.callmatrix("screenoff")}>Screen Off </Button></Col></Row>
            </Container>
        );
    }
}
export default Home;