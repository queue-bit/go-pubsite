# About

This is a Go program to generate my site, I wanted something that allowed me to have a clean documentation repo with minimal configuration files included.

This is an MVP for my specific use and has a fairly large backlog of technical debt, it is not intended for use by others. 

**If you are going to use this, please either fork it or use the version tagging as future updates _will_ break things. [I strongly suggest you use Hugo instead.](https://gohugo.io/)**

- [About](#about)
- [Directories](#directories)
  - [Content Directory --> */content/*](#content-directory----content)
    - [Config Directory --> */content/.config/*](#config-directory----contentconfig)
    - [Section Directories --> */content/1_section-name/*](#section-directories----content1_section-name)
    - [Category Directories --> */content/1_section-name/_category-name*](#category-directories----content1_section-name_category-name)
    - [Media Directory --> */content/_media*](#media-directory----content_media)
  - [Out Directory --> */out/*](#out-directory----out)
  - [Template Directory --> */templates/*](#template-directory----templates)
- [Files](#files)
  - [/content/index.md](#contentindexmd)
  - [/content/.config/config.yaml](#contentconfigconfigyaml)
  - [Ignored Files](#ignored-files)
- [Markdown Content Processing](#markdown-content-processing)
  - [Frontmatter](#frontmatter)
  - [Table of Contents](#table-of-contents)
  - [Diagrams](#diagrams)
  - [Mixed Markdown and HTML](#mixed-markdown-and-html)


# Directories

## Content Directory --> */content/*

You might notice that the `content` directory does not exist in this repo, the directory is stored in [another repo](https://github.com/queue-bit/queue-bit.github.io) that has a github action which pulls in (well, checks out) this repo to build the site.

The `content` directory needs to be of this structure:

```
./
│
├── content/
│   ├── index.md
│   ├── .config
│   |    └── config.yaml
│   ├── _media
│   |    ├── image-file.jpg
│   |    ├── pdf-file.pdf
│   |    └── etc.
│   ├── 1_first-section
│   |    ├── first-category
│   |    |   ├── first-markdown-file.md
│   |    |   └── second-markdown-file.md
│   |    └── second-category
│   |        └── first-markdown-file.md
│   ├── 2_second-section
│   |    └── first-category
│   |        └── first-markdown-file.md
│   └── 3_third-section
│        └── first-category
│            ├── first-markdown-file.md
│            └── second-markdown-file.md
│
└── README.md

```

### Config Directory --> */content/.config/*

Contains the [config.yaml](#configyaml) file, this is the only configuration file for the site.


### Section Directories --> */content/1_section-name/*

Sections show up in the Nav as top-level items, they must always start with a number followed by an underscore (eg. `1_`), the number determines it's position in the menu.

Everything after the number and underscore becomes the section name (dashes are removed and title case is applied).

Example:

`/content/1_first-section/` would be the first section and show up directly beside the Home button in the navigation with the label: `First Section`

### Category Directories --> */content/1_section-name/_category-name*

Categories show up in the Nav under the top-level items, they must always start with a underscore `_` and be inside a Section directory. 

The category name is set by removing the dashes and underscores and applying a title case to the directory name.

Example:

`/content/1_first-section/_first-category` would be listed under the navigation for 1_first-section and be labelled `First Category`

### Media Directory --> */content/_media*

Contains any non-markdown files you want to include in documents or as attachments. This includes things like images, pdf files, etc.


## Out Directory --> */out/*

The `out` directory is where the generated HTML pages, template artifacts (css, js, etc), and anything in _media are stored.

Since the github action creates a fresh environment each time it's run, the program does not automatically delete files from previous runs.

## Template Directory --> */templates/*

I only have use for one template, this directory contains a modified version of the ['txt' template created by HTML5UP](https://html5up.net/txt).

A second template could be added and called by setting `templatename` in the config.yaml file to the directory name of the new template, see the existing template for an idea on how to configure that.

# Files
## /content/index.md

`/content/index.md` contains the homepage text.

## /content/.config/config.yaml

By using the directory names to set Sections and Categories we eliminate the need for separate configuration files to set the taxonomy. The only configuration file we need `/content/.config/config.yaml` sets the overall site metadata:


```
title:          The site title
domain:         The domain of the site (www.example.com)
baseurl:        The base url of the site (http://www.example.com)
templatename:   The template name/directory (currently only supports "txt")
email:          Your email address (caution, this displays on the site)
github:         Link to your github account (https://github.com/example-user)
linkedin:       Link to your linkedin account (https://www.linkedin.com/in/example-user)
twitter:        Link to your twitter account (https://twitter.com/example-user)
analytics:      HTML of your analytics tags
ogtype:         Default OpenGraph type, can be overridden by article pages via frontmatter
author:         Default author, can be overridden by article pages via frontmatter
ogimage:        Default OpenGraph image, can be overridden by article pages via frontmatter
faviconpath:    The relative path to the favicon
```

## Ignored Files

When the content directory is processed, filenames and directory names that contain the following are ignored: 

- .yaml
- .git
- .config
- README.md


# Markdown Content Processing

## Frontmatter

Frontmatter is supported, defined at the top of the document between three dashes `---`. Currently supported tags:

1. title
1. excerpt
1. tags (future use)

Note that the excerpt is displayed on content pages between the breadcrumb navigation and TOC.

Example:
```
---
title:        "A sample title"
intro:        "Excerpt I want to display between the breadcrumbs and the TOC"
tags:         "Comma separated list of tags, used in OpenGraph metadata on the site"
ogtype:       "OpenGraph type for the page"
author:       "Author for the page, used in OpenGraph metadata"
description:  "Description for the page, used in metadata and OpenGraph metadata"
date:         "Publish date for the page, used in OpenGraph metadata"
ogimage:      "OpenGraph image for the page, used in OpenGraph metadata"
---
```

## Table of Contents

The program will automatically generate a Table of Contents for markdown files that have more than two headings.

## Diagrams

The program supports [mermaid.js](https://mermaid-js.github.io/mermaid/) diagrams in the markdown files, to use them you need to encapsulate them with three backticks and the word mermaid:

````
```mermaid
graph TD;
    A-->B;
    A-->C;
    B-->D;
    C-->D;
```
````

## Mixed Markdown and HTML

HTML is allowed in the markdown files and will be passed along as-is.




