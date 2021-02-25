import React from 'react';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Form from 'react-bootstrap/Form';
import 'bootstrap/dist/css/bootstrap.min.css';

var BACKEND = "http://" + window.location.host

class Home extends React.Component {
    constructor(props) {
        super(props);
        this.state = { "disablerChecked": false };
    }
    callmatrix(path) {
        console.log(`Calling matrix API ${path}`)
        fetch(`${BACKEND}/api/${path}`, {
            method: "GET",
            mode: "cors",
        });
    }
    async componentDidMount() {
        const resp = await fetch(`${BACKEND}/api/status`,
            {
                method: "GET",
                mode: "cors",
            }
        );

        const status = await resp.text();

        if (resp.ok) {
            if (status === "true") {
                console.log("board is enabled", status)
                this.setState({ "disablerChecked": true })
            } else {
                console.log("board is disabled", status)
                this.setState({ "disablerChecked": false })
            }
        }
    }

    handleSwitch = () => {
        if (!this.state.disablerChecked) {
            console.log("enabling board")
            this.callmatrix("screenon")
        } else {
            console.log("disabling board")
            this.callmatrix("screenoff")
        }
        this.setState({ "disablerChecked": !this.state.disablerChecked })
    }
    render() {
        return (
            <Container fluid>
                <Row className="text-center">
                    <Col>
                        <Form.Switch id="enabler" label="Screen On/Off" checked={this.state.disablerChecked} onChange={this.handleSwitch} />
                    </Col>
                </Row>
            </Container>
        );
    }
}
export default Home;