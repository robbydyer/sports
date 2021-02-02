import React from 'react';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import 'bootstrap/dist/css/bootstrap.min.css';

const MATRIX = "http://matrix.local:8080"

class Home extends React.Component {
    render() {
        return (
            <Container fluid>
                <Row className="text-center"><Col><Button onClick={this.screenOn}>Screen On </Button></Col></Row>
                <Row className="text-center"><Col><Button onClick={this.screenOff}>Screen Off </Button></Col></Row>
            </Container>
        );
    }
}
export default Home;