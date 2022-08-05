import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Card from 'react-bootstrap/Card';
import Image from 'react-bootstrap/Image';
import Home from './Home.js';
import Sport from './Sport.js';
import Racing from './Racing.js';
import ImageBoard from './ImageBoard.js';
import BasicBoard from './BasicBoard';
import Weather from './Weather.js';
import Accordion from 'react-bootstrap/Accordion';
import { LogoSrc } from './Logo.js';

const styles = {
    row: {
        marginTop: 10
    },
    col: {
        paddingTop: '20px'
    }
}

const card_border = "18rem"

class All extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            "sync": Date.now(),
        }
    }

    doSync = () => {
        console.log("All page updating sync time")
        this.setState({ "sync": Date.now() })
    }

    render() {
        var sports = ["ncaaf", "nhl", "mlb", "ncaam", "nfl", "nba", "mls", "epl", "dfl", "dfb", "uefa"].map((sport) =>
            <Accordion.Item eventKey={sport}>
                <Accordion.Header>
                    <Image src={LogoSrc(sport)} style={{ height: '100px', width: 'auto' }} />
                </Accordion.Header>
                <Accordion.Body>
                    <Card style={{ width: { card_border } }}>
                        <Sport sport={sport} id={sport} key={sport + this.state.sync} doSync={this.doSync} />
                    </Card>
                </Accordion.Body>
            </Accordion.Item>
        );
        var racing = ["f1", "irl"].map((sport) =>
            <Col>
                <Accordion.Item eventKey={sport}>
                    <Accordion.Header><Image src={LogoSrc(sport)} style={{ height: '100px', width: 'auto' }} /></Accordion.Header>
                    <Accordion.Body>
                        <Card style={{ width: { card_border } }}>
                            <Racing sport={sport} id={sport} key={sport + this.state.sync} doSync={this.doSync} />
                        </Card>
                    </Accordion.Body>
                </Accordion.Item>
            </Col>
        );
        return (
            <Container fluid>
                <Row className="justify-content-space-between" sm={1} lg={2} xl={3} style={styles.row}>
                    <Col>
                        <Card style={{ width: { card_border } }}>
                            <Home doSync={this.doSync} key={"home" + this.state.sync} />
                        </Card>
                    </Col>
                </Row>
                <Row className="justify-content-space-between" sm={1} lg={2} xl={3} style={styles.row}>
                    <Col>
                        <Accordion>
                            {sports}

                            <Accordion.Item eventKey="pga">
                                <Accordion.Header><Image src={LogoSrc("pga")} style={{ height: '100px', width: 'auto' }} fluid /></Accordion.Header>
                                <Accordion.Body>
                                    <Card style={{ width: { card_border } }}>
                                        <BasicBoard id="pga" name="pga" doSync={this.doSync} key={"pga" + this.state.sync} path="stat/pga" />
                                    </Card>
                                </Accordion.Body>
                            </Accordion.Item>
                            <Accordion.Item eventKey="weather">
                                <Accordion.Header><Image src={LogoSrc("weather")} style={{ height: '100px', width: 'auto' }} fluid /></Accordion.Header>
                                <Accordion.Body>
                                    <Card style={{ width: { card_border } }}>
                                        <Weather id="weatherboard" doSync={this.doSync} key={"weather" + this.state.sync} />
                                    </Card>
                                </Accordion.Body>
                            </Accordion.Item>
                            <Accordion.Item eventKey="imgboard">
                                <Accordion.Header><Image src={LogoSrc("img")} style={{ height: '100px', width: 'auto' }} fluid /></Accordion.Header>
                                <Accordion.Body>
                                    <Card style={{ width: { card_border } }}>
                                        <ImageBoard id="imgboard" doSync={this.doSync} key={"img" + this.state.sync} />
                                    </Card>
                                </Accordion.Body>
                            </Accordion.Item>
                            <Accordion.Item eventKey="stocks">
                                <Accordion.Header><Image src={LogoSrc("stocks")} style={{ height: '100px', width: 'auto' }} fluid /></Accordion.Header>
                                <Accordion.Body>
                                    <Card style={{ width: { card_border } }}>
                                        <BasicBoard id="stocks" name="stocks" doSync={this.doSync} key={"stocks" + this.state.sync} />
                                    </Card>
                                </Accordion.Body>
                            </Accordion.Item>
                            <Accordion.Item eventKey="clock">
                                <Accordion.Header><Image src={LogoSrc("clock")} style={{ height: '100px', width: 'auto' }} fluid /></Accordion.Header>
                                <Accordion.Body>
                                    <Card style={{ width: { card_border } }}>
                                        <BasicBoard id="clock" name="clock" doSync={this.doSync} key={"clock" + this.state.sync} />
                                    </Card>
                                </Accordion.Body>
                            </Accordion.Item>
                            <Accordion.Item eventKey="gcal">
                                <Accordion.Header><Image src={LogoSrc("gcal")} style={{ height: '100px', width: 'auto' }} fluid /></Accordion.Header>
                                <Accordion.Body>
                                    <Card style={{ width: { card_border } }}>
                                        <BasicBoard id="gcal" name="gcal" doSync={this.doSync} key={"gcal" + this.state.sync} />
                                    </Card>
                                </Accordion.Body>
                            </Accordion.Item>
                            <Accordion.Item eventKey="sys">
                                <Accordion.Header><Image src={LogoSrc("sys")} style={{ height: '100px', width: 'auto' }} fluid /></Accordion.Header>
                                <Accordion.Body>
                                    <Card style={{ width: { card_border } }}>
                                        <BasicBoard id="sys" name="sys" doSync={this.doSync} key={"sys" + this.state.sync} />
                                    </Card>
                                </Accordion.Body>
                            </Accordion.Item>
                            {racing}
                        </Accordion>
                    </Col>
                </Row>

            </Container>
        )
    }
}

export default All;