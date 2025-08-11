<!-- Improved compatibility of back to top link: See: https://github.com/othneildrew/Best-README-Template/pull/73 -->
<a id="readme-top"></a>
<!--
*** Thanks for checking out the Best-README-Template. If you have a suggestion
*** that would make this better, please fork the repo and create a pull request
*** or simply open an issue with the tag "enhancement".
*** Don't forget to give the project a star!
*** Thanks again! Now go create something AMAZING! :D
-->



<!-- PROJECT SHIELDS -->
<!--
*** I'm using markdown "reference style" links for readability.
*** Reference links are enclosed in brackets [ ] instead of parentheses ( ).
*** See the bottom of this document for the declaration of the reference variables
*** for contributors-url, forks-url, etc. This is an optional, concise syntax you may use.
*** https://www.markdownguide.org/basic-syntax/#reference-style-links
-->
[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![project_license][license-shield]][license-url]
[![LinkedIn][linkedin-shield]][linkedin-url]



<!-- PROJECT LOGO -->
<br />
<div align="center">
  <a href="https://github.com/CleverNamesTaken/Ask">
    <!--<img src="images/logo.png" alt="Logo" width="80" height="80"> -->
  </a>

<h3 align="center">Ask</h3>

  <p align="center">
    Ask (Aggregated Snippet Knowledgebase) is a commandline tool written in Golang to help red team operators save, improve, and share code snippets.
    <br />
    <a href="https://github.com/CleverNamesTaken/Ask"><strong>Explore the docs »</strong></a>
    <br />
    <br />
    <!-- <a href="https://github.com/CleverNamesTaken/Ask">View Demo</a> -->
    &middot;
    <a href="https://github.com/CleverNamesTaken/Ask/issues/new?labels=bug&template=bug-report---.md">Report Bug</a>
    &middot;
    <a href="https://github.com/CleverNamesTaken/Ask/issues/new?labels=enhancement&template=feature-request---.md">Request Feature</a>
  </p>
</div>



<!-- TABLE OF CONTENTS -->
<details>
  <summary>Table of Contents</summary>
  <ol>
    <li>
      <a href="#about-the-project">About The Project</a>
      <ul>
        <li><a href="#built-with">Built With</a></li>
      </ul>
    </li>
    <li>
      <a href="#getting-started">Getting Started</a>
      <ul>
        <li><a href="#prerequisites">Prerequisites</a></li>
        <li><a href="#installation">Installation</a></li>
      </ul>
    </li>
    <li><a href="#usage">Usage</a></li>
    <li><a href="#roadmap">Roadmap</a></li>
    <li><a href="#contributing">Contributing</a></li>
    <li><a href="#license">License</a></li>
    <li><a href="#contact">Contact</a></li>
    <li><a href="#acknowledgments">Acknowledgments</a></li>
  </ol>
</details>



<!-- ABOUT THE PROJECT -->
## About The Project

What is this? This project is an attempt to solve the following problems that people who conduct complex network operations might face:
1. Broad toolsets that use a variety of hard-to-remember syntaxes.
2. Implement version control on code snippets to keep track of incremental improvements
3. Standardize code snippets and capture metadata about the snippets to try to dummy-proof them and allow others to adopt them

The philosophy behind the project is that perfection is unattainable but we can always strive for clear documentation and continuous improvement.

I wrote the tool because I noticed that operators used different snippet engines and I wanted the ability to standardize and share snippets.

Ask uses a database to version control snippets.  It can use either mysql or sqlite depending on the configuration file you use.

Ask makes use of a TUI to help you write structured snippets that contain metadata that other snippet or command lookup tools might not have.

<p align="right">(<a href="#readme-top">back to top</a>)</p>



### Built With

* [![Golang][go.dev]][https://go.dev]

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- GETTING STARTED -->
## Getting Started

Download or compile the ask binary.

### Prerequisites

None.  If you choose to serve your database of mysql, then you will need to write your configuration file with the IP, port, and credentials.

`ask create conf`

### Installation

None.

If you want to use mysql:

```
sudo apt update
sudo apt install mariadb-server -y
systemctl start mysql
mysqladmin -u root password 'newpassword'
```

## Usage

```
ask create template
    #Use the quickstart option unless you already have a mysql server you'd like to use.
ask add example.txt
ask ls
ask render example
ask edit example
```

<!-- Use this space to show useful examples of how a project can be used. Additional screenshots, code examples and demos work well in this space. You may also link to more resources. -->

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- ROADMAP -->
## Roadmap

- [ ] Create a default zip archive with some basic snippets
- [ ] Create tutorial on how to use variable yamls to make rendering easier
- [ ] vscode snippet format for rendering
- [ ] implement tagging system for searching
    - [ ] tab completion for tags in TUI
- [ ] interactive metasploit-like console for searching
- [ ] fuzzy searching
- [ ] implement heirarchy
    - [ ]	add suggestions from existing heirarchy in snippet forms.
    - [ ]	in the console, allow use with tab completion of available snippets to discover more

See the [open issues](https://github.com/CleverNamesTaken/Ask/issues) for a full list of proposed features (and known issues).

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- CONTRIBUTING -->
## Contributing

Contributions are what make the open source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

If you have a suggestion that would make this better, please fork the repo and create a pull request. You can also simply open an issue with the tag "enhancement".
Don't forget to give the project a star! Thanks again!

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Top contributors:

<a href="https://github.com/CleverNamesTaken/Ask/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=CleverNamesTaken/Ask" alt="contrib.rocks image" />
</a>



<!-- LICENSE -->
## License

Distributed under the MIT License. See `LICENSE.txt` for more information.

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- CONTACT -->
## Contact

Project Link: [https://github.com/CleverNamesTaken/Ask](https://github.com/CleverNamesTaken/Ask)

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- ACKNOWLEDGMENTS -->
## Acknowledgments

This tool was inspired by other great projects:

* [RTFM](https://github.com/leostat/rtfm)
* [Arsenal](https://github.com/Orange-Cyberdefense/arsenal)

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- MARKDOWN LINKS & IMAGES -->
<!-- https://www.markdownguide.org/basic-syntax/#reference-style-links -->
[contributors-shield]: https://img.shields.io/github/contributors/CleverNamesTaken/Ask.svg?style=for-the-badge
[contributors-url]: https://github.com/CleverNamesTaken/Ask/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/CleverNamesTaken/Ask.svg?style=for-the-badge
[forks-url]: https://github.com/CleverNamesTaken/Ask/network/members
[stars-shield]: https://img.shields.io/github/stars/CleverNamesTaken/Ask.svg?style=for-the-badge
[stars-url]: https://github.com/CleverNamesTaken/Ask/stargazers
[issues-shield]: https://img.shields.io/github/issues/CleverNamesTaken/Ask.svg?style=for-the-badge
[issues-url]: https://github.com/CleverNamesTaken/Ask/issues
[license-shield]: https://img.shields.io/github/license/CleverNamesTaken/Ask.svg?style=for-the-badge
[license-url]: https://github.com/CleverNamesTaken/Ask/blob/master/LICENSE.txt
