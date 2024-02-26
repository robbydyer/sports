import React from 'react';
import 'bootstrap/dist/css/bootstrap.min.css';
import Container from 'react-bootstrap/Container';
import Navbar from 'react-bootstrap/Navbar';
import Nav from 'react-bootstrap/Nav';
import NavDropDown from 'react-bootstrap/NavDropdown';
import { Link } from 'react-router-dom';
import { GetVersion } from './util';
import { useNavigate } from 'react-router';

export const withRouter = (Component) => {
    const Wrapper = (props) => {
        const history = useNavigate();
        return <Component history={history} {...props} />
    }
    return Wrapper;
}

class TopNav extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            "version": "",
        }
    }
    componentDidMount() {
        if (this.state.version === "") {
            console.log("fetching version")
            GetVersion((val) => {
                this.setState({
                    "version": val,
                })
            })
        }
    }
    render() {
        return (
            <Container fluid>
                <Navbar expand="sm" bg="dark" variant="dark" hidden={window.location.pathname === "/board" ? true : false}>
                    <Navbar.Brand>SportsMatrix</Navbar.Brand>
                    <Navbar.Toggle aria-controls="basic-navbar-nav"></Navbar.Toggle>
                    <Navbar.Collapse id="basic-navbar-nav">
                        <Nav className="mr-auto">
                            <Nav.Link as={Link} to="/">Home</Nav.Link>
                            <NavDropDown bg="dark" variant="dark" title="Sports" id="sports-drop">

                                <NavDropDown.Item as={Link} to="/mlb">MLB</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/mls">MLS</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/nba">NBA</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/ncaaf">NCAAF</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/nhl">NHL</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/ncaam">NCAA Men's Basketball</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/nfl">NFL</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/pga">PGA</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/epl">EPL</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/dfl">DFL</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/dfb">DFB</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/uefa">UEFA</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/fifa">FIFA</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/ncaaw">NCAA Women's Basketball</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/wnba">WNBA</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/ligue">Ligue 1</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/seriea">Serie A</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/laliga">La Liga</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/xfl">XFL</NavDropDown.Item>
                            </NavDropDown>
                            <Nav.Link as={Link} to="/stocks">Stocks</Nav.Link>
                            <Nav.Link as={Link} to="/weather">Weather</Nav.Link>
                            <NavDropDown bg="dark" variant="dark" title="Racing" id="racing-drop">
                                <NavDropDown.Item as={Link} to="/f1">F1</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/irl">IndyCar</NavDropDown.Item>
                            </NavDropDown>
                            <NavDropDown bg="dark" variant="dark" title="Misc" id="misc-drop">
                                <NavDropDown.Item as={Link} to="/img">Image Board</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/clock">Clock</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/gcal">Calendar</NavDropDown.Item>
                                <NavDropDown.Item as={Link} to="/sys">System Info</NavDropDown.Item>
                            </NavDropDown>
                            <Nav.Link as={Link} to="/docs">API Docs</Nav.Link>
                            <Nav.Link as={Link} to="/board">Live Board</Nav.Link>
                        </Nav>
                        <Navbar.Text>{this.state.version}</Navbar.Text>
                    </Navbar.Collapse>
                </Navbar>
            </Container >
        );
    }
}

export default withRouter(TopNav);