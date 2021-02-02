import React, { useState } from 'react';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import 'bootstrap/dist/css/bootstrap.min.css';

class App extends React.Component {
  render() {
    return (
      <Container fluid>
        <Row><Col><Button href="http://matrix.local:8080/screenoff">Screen Off</Button></Col></Row>
        <Row><Col><Button href="http://matrix.local:8080/screenon">Screen On</Button></Col></Row>
      </Container>
    );
  }
}
export default App;