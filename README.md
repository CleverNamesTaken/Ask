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
3. Standardize code snippets and capture metadata about the snippets to try to dummy-proof them 
4. Share code snippets with others that use a variety of workflows (VSCode, vim, straight bash, Obsidian)

The philosophy behind the project is that perfection is unattainable but we can always strive for clear documentation and continuous improvement.

I wrote the tool because I noticed that operators used different snippet engines and I wanted the ability to standardize and share snippets.

Ask uses a database to version control snippets.  It can use either mysql or sqlite depending on the configuration file you use.

Ask makes use of a TUI to help you write structured snippets that contain metadata that other snippet or command lookup tools might not have.

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!--
### Built With

* [![Golang][go.dev]][https://go.dev]

<p align="right">(<a href="#readme-top">back to top</a>)</p>
-->



<!-- GETTING STARTED -->
## Getting Started

Download or compile the ask binary.

### Prerequisites

None.  If you choose to serve your database with mysql, then you will need to write your configuration file with the IP, port, and credentials.

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
  Ask (Aggregated Snippet Knowledgebase) by the OtherDave (https://github.com/clevernamestaken) is a CLI tool written in
  golang to save, edit, and share code snippets for terminal-based collaborative operations using databases and         
  structured TUI forms.                                                                                                 
                                                                                                                        
  EXAMPLES                                                                                                              
  ask create config                                                                                                     
      #Create the config file to start interacting with a database or create a new one                                  
                                                                                                                        
  ask create template                                                                                                   
      #Create an example snippet file                                                                                   
                                                                                                                        
  ask add example.txt                                                                                                   
      #Ingest the snippet file into the database                                                                        
                                                                                                                        
  ask cat example                                                                                                       
      #Look at the raw snippet text for the snippet named "example"                                                     
                                                                                                                        
  ask edit example                                                                                                      
      #Edit a snippet called "example"                                                                                  
                                                                                                                        
  ask ls                                                                                                                
      #Examine which snippets are in the database                                                                       
                                                                                                                        
  ask ls mple                                                                                                           
      #Examine which snippets in the database have a name containing the string "mple"                                  
                                                                                                                        
  ask render example                                                                                                    
      #Render the snippet named "example" to stdout after being prompted to fill out the variables                      
                                                                                                                        
  ask render text --all --outdir ./text_snippets                                                                        
      #Dump the entire database in text format to a directory called ./text_snippets                                    
                                                                                                                        
  ask render create zip --outfile archive.zip                                                                           
      #Create an archive of the database                                                                                
                                                                                                                        
  ask rm --prune                                                                                                        
      #Remove all of the outdated snippets from the database                                                            
                                                                                                                        
  ask  add archive.zip                                                                                                  
      #Add an archived zip to an existing database.                                                                     
                                                                                                                        
  ask console                                                                                                           
      #Start a metasploit-like console to search for and use snippets                                                   
                                                                                                                        
         
  USAGE  
         
    ask [command] [subcommand] [--flags]  
            
  COMMANDS  
            
    add                         Add new snippets to the database
    browse                      Browse snippets in the database. Aliased to 'ls'
    completion [command]        Generate the autocompletion script for the specified shell
    console                     Start the ask console
    create [command]            Create config file, snippet template files, zip archive or the database schema
    edit                        Edit an existing snippet
    help [command]              Help about any command
    remove [--flags]            Remove snippets from the database. Aliased to 'rm'
    render [command] [--flags]  Render snippets
    show                        Take a quick peek at a raw snippet, excluding the metadata. Aliased to 'cat'
         
  FLAGS  
         
    -c --config                 Config file (default is $HOME/.config/ask/config.yaml)
    -d --debug                  Enable debug output
    -h --help                   Help for ask
    -v --version                Version for ask
```

<!-- Use this space to show useful examples of how a project can be used. Additional screenshots, code examples and demos work well in this space. You may also link to more resources. -->

### Overthewire example

```
git clone git@github.com:CleverNamesTaken/Ask.git
go build -o ask
chmod +x ask
./ask add archive.zip
2
    #Create the sqlite database just to start going
./ask render bandit0
./ask render bandit1
./ask render bandit2 -l examples/bandit2.yaml
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- ROADMAP -->
## Roadmap

- [x] Create a default zip archive with some basic snippets
- [x] Create tutorial on how to use variable yamls to make rendering easier
- [x] vscode snippet format for rendering
- [x] implement tagging system for searching
    - [x] tab completion for tags in TUI
- [x] interactive metasploit-like console for searching
    - [x] Set flags for rendering, output file
    - [ ] set global variables
- [ ] fuzzy searching
- [ ] implement heirarchy
    - [ ]	add suggestions from existing heirarchy in snippet forms.
    - [x]	in the console, allow use with tab completion of available snippets to discover more
- [ ] Unit testing
- [ ] Render output to system clipboard
- [x] Render to Templater-Obsidian format

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

## Change log

- v0.1.0 : First release
- v0.1.1 : Fixed the overwrite of a config file and added some aliases.
- v0.2.0 : VSCode snippet rendering
- v0.3.0 : Ask console
- v0.3.1 : console flags, tab completion
- v0.3.2 : Added choices for default values in ultisnips and vscode rendering.  Also tab completion for default value and choices in console.
- v0.4.0 : Improved vscode rendering.  Added tagging system.  Removed clipboard rendering (for now)
- v0.4.1 : Improved vscode rendering.  Added --all tag for browsing. Cleaned up tmp directory after creating temp files.  Allow tag or tags for search field
- v0.5.0 : Added Obsidian template stuff.  Fixed more temp files left over.



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
