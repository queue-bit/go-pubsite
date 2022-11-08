// http-server . -p 8000
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"

	mermaid "github.com/abhinav/goldmark-mermaid"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"golang.org/x/exp/slices"
	newhtml "golang.org/x/net/html"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Title         string        `yaml:"title"`
	Domain        string        `yaml:"domain"`
	Email         string        `yaml:"email"`
	Github        string        `yaml:"github"`
	Linkedin      string        `yaml:"linkedin"`
	Twitter       string        `yaml:"twitter"`
	TemplateName  string        `yaml:"templatename"`
	BaseURL       string        `yaml:"baseurl"`
	Analytics     template.HTML `yaml:"analytics"`
	DefaultOgType string        `yaml:"ogtype"`
	Author        string        `yaml:"author"`
	OgImage       string        `yaml:"ogimage"`
	FavIconPath   string        `yaml:"faviconpath"`
}

type Page struct {
	Title       string
	Content     template.HTML
	Path        string
	SiteRoot    string
	Category    string
	Section     string
	Index       int
	Nav         template.HTML
	Intro       template.HTML
	Analytics   template.HTML
	Description template.HTML
	OgType      string
	Author      string
	Url         string
	Date        string
	OgImage     string
	Tags        string
}

type Category struct {
	Title  string
	Parent string
	Crumb  string
}

type Section struct {
	Title string
	Index int
	Crumb string
}

type Paths struct {
	Content      string
	Output       string
	Template     string
	Asset        string
	TemplateSub  string
	TemplateRoot string
}

type TopNav struct {
	Nav template.HTML
}

//Setup some site metadata
func siteMeta(configFile string) Config {

	config := Config{}

	file, err := os.Open(configFile)
	if err != nil {
		panic("Could not open configuration file")
	}
	defer file.Close()

	// Init new YAML decode
	d := yaml.NewDecoder(file)

	// Start YAML decoding from file
	if err := d.Decode(&config); err != nil {
		return Config{}
	}
	return config
}

//Setup some default paths
func paths(currentDirectory string, siteConfig Config) Paths {
	currentTemplate := "/templates/" + siteConfig.TemplateName
	contentDirectory := currentDirectory + "/content"
	outputDirectory := strings.Replace(contentDirectory, "content", "out", 1)
	templateDirectory := currentDirectory + currentTemplate + "/base/"
	assetDirectory := currentDirectory + currentTemplate + "/assets/"

	return Paths{
		Content:      contentDirectory,
		Output:       outputDirectory,
		Template:     templateDirectory,
		Asset:        assetDirectory,
		TemplateSub:  currentTemplate + "/base/",
		TemplateRoot: currentTemplate,
	}
}

/*	Content is split up by directories
	Top-level navigation (shows on menus) are stored in directories named #_name (e.g. 1_about) and are called 'Sections'*/
func pageSection(workingFile string) Section {
	var section string
	var index int
	var sectionRe = regexp.MustCompile(`(\d{1,5})_(.+?)\/`)

	sectionMatches := sectionRe.FindStringSubmatch(workingFile)
	if len(sectionMatches) >= 1 {
		//section = fmt.Sprintf("%s", sectionMatches[2])
		section = sectionMatches[2]
		index, _ = strconv.Atoi(sectionMatches[1])
	} else {
		section = ""
		index = 0
	}

	return Section{
		Index: index,
		Title: strings.Replace(cases.Title(language.Und).String(section), "-", " ", 1),
		Crumb: section,
	}
}

/*	Content is split up by directories
	Second-level navigation (shows on category pages) are stored in directories named _name (e.g. _work) and are called 'Categories'*/
func pageCategory(workingFile string) Category {
	var category string
	var parentCategory string
	var categoryRe = regexp.MustCompile(`_(.+?)\/`)

	categoryMatches := categoryRe.FindAllStringSubmatch(workingFile, -1)
	if len(categoryMatches) == 2 {
		parentCategory = fmt.Sprintf("%s", categoryMatches[0])
		category = fmt.Sprintf("%s", categoryMatches[1])
	} else if len(categoryMatches) == 1 {
		category = fmt.Sprintf("%s", categoryMatches[0])
	} else {
		category = ""
	}

	return Category{
		Title:  strings.Replace(cases.Title(language.Und).String(category), "-", "", 1),
		Parent: parentCategory,
		Crumb:  category,
	}
}

func parsePage(workingFile string, paths Paths, siteConfig Config) Page {

	outFile := strings.Replace(workingFile, "content", "out", 1)
	outFile = strings.Replace(outFile, ".md", ".html", 1)

	content, err := ioutil.ReadFile(workingFile)
	if err != nil {
		log.Print("ERROR: ", err, " workingFile: ", workingFile, " in parsePage(string,paths,config)->ReadFile")
	}

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			meta.Meta,
			extension.TaskList,
			&mermaid.Extender{},
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			html.WithUnsafe(),
		),
	)

	var buf bytes.Buffer
	context := parser.NewContext()
	if err := md.Convert(content, &buf, parser.WithContext(context)); err != nil {
		panic(err)
	}
	metaData := meta.Get(context)
	title := metaData["title"].(string)
	pageCategory := pageCategory(workingFile)
	pageSection := pageSection(workingFile)
	pageIntro := metaData["intro"].(string)
	pageDescription := metaData["description"].(string)

	var ogImage string
	if metaData["ogimage"] == nil {
		ogImage = siteConfig.BaseURL + "/media/" + siteConfig.OgImage
	} else {
		ogImage = siteConfig.BaseURL + "/media/" + metaData["ogimage"].(string)
	}

	var ogType string
	if metaData["ogtype"] == nil {
		ogType = siteConfig.DefaultOgType
	} else {
		ogType = metaData["ogtype"].(string)
	}

	var tags string
	if metaData["tags"] == nil {
		tags = ""
	} else {
		tags = metaData["tags"].(string)
	}

	var articleDate string
	if metaData["date"] == "" {
		articleDate = ""
	} else {
		articleDate = metaData["date"].(string)
	}

	outFile = strings.Replace(outFile, strconv.Itoa(pageSection.Index)+"_", "", 1)
	outFile = strings.Replace(outFile, "_", "", 1)

	var categoryCrumb string
	categoryCrumb = pageCategory.Crumb[strings.LastIndex(pageCategory.Crumb, " ")+1:]
	categoryCrumb = strings.TrimRight(categoryCrumb, "]")

	pageUrl := siteConfig.BaseURL + strings.Split(outFile, "/out")[1]

	var pageNav string
	if pageSection.Crumb == "" {
		pageNav = ""
	} else {
		pageNav = "<a href=\"" + siteConfig.BaseURL + "\">Home</a> // <a href=\"" + siteConfig.BaseURL + "/" + pageSection.Crumb + "\">" + strings.Replace(cases.Title(language.Und).String(pageSection.Crumb), "-", " ", 1) + "</a>" + " // " + "<a href=\"" + siteConfig.BaseURL + "/" + pageSection.Crumb + "/" + categoryCrumb + "\">" + strings.Replace(cases.Title(language.Und).String(categoryCrumb), "-", " ", 1) + "</a> // " + title
	}

	return Page{
		Title:       title,
		Content:     template.HTML(buf.String()),
		Path:        outFile,
		Category:    pageCategory.Title,
		Section:     pageSection.Title,
		Index:       pageSection.Index,
		SiteRoot:    siteConfig.BaseURL,
		Nav:         template.HTML(pageNav),
		Intro:       template.HTML(pageIntro),
		Description: template.HTML(pageDescription),
		Analytics:   siteConfig.Analytics,
		Author:      siteConfig.Author,
		OgType:      ogType,
		Url:         strings.Replace(pageUrl, ".html", "", 1),
		Date:        articleDate,
		OgImage:     ogImage,
		Tags:        tags,
	}

}

func createDirectory(createPath string) {
	_, err := os.Stat(createPath)
	if os.IsNotExist(err) {
		errorDir := os.MkdirAll(createPath, 0755)
		if errorDir != nil {
			log.Print("ERROR: ", errorDir, " with MkdirAll(", createPath, ") in createDirectory(string)")
		}
	}
}

func copyFile(currentFile string, outPath string) {
	originalFile, err := os.Open(currentFile)
	if err != nil {
		log.Fatal("FATAL: ", err, " Could not open currentFile in copyFile(string,string)")
	}

	newFile, err := os.Create(outPath)
	if err != nil {
		log.Fatal("FATAL: ", err, " Could not create newFile ", outPath, " in copyFile(string,string)")
	}

	_, err = io.Copy(newFile, originalFile)

	if err != nil {
		log.Fatal("FATAL: ", err, " Could not copy ", originalFile, " to ", newFile.Name(), " in copyFile(string,string)")
	}

	err = newFile.Sync()
	if err != nil {
		log.Fatal("FATAL: ", err, " Could not Sync file: ", newFile.Name(), " in copyFile(string,string)")
	}

	defer originalFile.Close()
	defer newFile.Close()

}

func createPage(currentPage Page, sections []Section, topNav template.HTML, siteConfig Config) {
	var templateFiles = []string{"header.html", "footer.html", "body.html", "base.html"}
	var allPaths []string

	file, err := os.Create(currentPage.Path)
	if err != nil {
		log.Fatal("FATAL: ", err, " Could not create ", currentPage.Path, " in createPage(Page,Section,template.HTML,Config)")
	}

	defer file.Close()

	for _, tmpl := range templateFiles {
		allPaths = append(allPaths, "./"+paths(currentPage.Path, siteConfig).TemplateSub+tmpl)
	}

	templates := template.Must(template.New("").ParseFiles(allPaths...))

	Toc := addToc(string(currentPage.Content), string(currentPage.Title))

	var processed bytes.Buffer
	err = templates.ExecuteTemplate(&processed, "Base", struct{ CurrentPage, SiteMetaData, TopNav, Toc, Sections interface{} }{currentPage, siteConfig, topNav, Toc, sections})

	if err != nil {
		log.Fatal("FATAL: ", err, " Could not ExecuteTemplate for ", currentPage.Path, " in createPage(Page,Section,template.HTML,Config")
	}

	f, _ := os.Create(currentPage.Path)
	w := bufio.NewWriter(f)
	w.WriteString(processed.String())
	w.Flush()
	defer f.Close()

}

func addToc(currentHtmlString string, currentTitle string) template.HTML {
	tokenizer := newhtml.NewTokenizer(strings.NewReader(currentHtmlString))
	var lastToc string
	var lastLevel int
	var tocLineItem string
	var level int
	var count int

	lastLevel = 0
	level = 0
	count = 0

	for {
		tocLineItem = ""
		tt := tokenizer.Next()

		if tt == newhtml.ErrorToken {
			if tokenizer.Err() == io.EOF {
				break
			}
			log.Print("ERROR: tokenizer: ", tokenizer.Err(), " in addToc")
			break
		}

		tag, hasAttr := tokenizer.TagName()

		if hasAttr {
			attrKey, attrValue, _ := tokenizer.TagAttr()
			tokenizer.Next() //Need to move it one to get the text value
			tocLinkItem := "<a href=\"#" + string(attrValue) + "\">" + string(tokenizer.Token().Data) + "</a>"

			if string(attrKey) == "id" {

				switch string(tag) {
				case "h2": // H1 is always the site title so we start at H2
					level = 1 // and set it to "1" to make it easier for me to understand later
				case "h3":
					level = 2
				case "h4":
					level = 3
				case "h5":
					level = 4
				case "h6":
					level = 5
				}

				lastLevel, tocLineItem = levels(level, lastLevel, tocLinkItem)

				if tocLineItem != "" {
					lastToc = lastToc + tocLineItem
					count++
				}

			}

		}

	}
	var closeTags string

	for i := lastLevel; i > 0; i-- {
		closeTags = closeTags + "<!--cbs--></li></ul>"
	}

	if count >= 3 {
		return template.HTML(lastToc + closeTags + "<br/>\n\n")
	} else {
		return template.HTML("")
	}

}

func levels(level int, lastLevel int, tocLinkItem string) (int, string) {
	//adapted from https://stackoverflow.com/a/4912737
	tocLineItem := ""
	closeTags := ""

	if level > lastLevel {
		tocLineItem = "<ul>"
	} else {
		closeTags = strings.Repeat("</li></ul>", lastLevel-level)
		closeTags = closeTags + "</li>"
	}

	tocLineItem = tocLineItem + closeTags + "<li>" + tocLinkItem
	lastLevel = level

	return lastLevel, tocLineItem + "\n"
}

func buildNavigation(sections []Section, categories []Category, pages []Page, paths Paths, siteConfig Config) (strings.Builder, []Page) {
	var parentCategory string
	var category string
	var categoryCrumb string
	var categoryPageHtml strings.Builder
	var sectionPageHtml strings.Builder
	var topNav strings.Builder

	for _, currentSection := range sections {
		if currentSection.Crumb == "" {
			continue
		}
		sectionPageHtml.Reset()
		sectionPageHtml.WriteString("<ul>")
		topNav.WriteString("<li>\n<a href=\"" + siteConfig.BaseURL + "/" + currentSection.Crumb + "\">" + strings.Replace(currentSection.Crumb, "-", " ", 1) + "</a>\n<ul>\n")
		for _, currentCategory := range categories {
			categoryPageHtml.Reset()
			parentCategory = currentCategory.Parent[strings.LastIndex(currentCategory.Parent, " ")+1:]
			parentCategory = strings.TrimRight(parentCategory, "]")

			if parentCategory == currentSection.Crumb {
				categoryCrumb = currentCategory.Crumb[strings.LastIndex(currentCategory.Crumb, " ")+1:]
				categoryCrumb = strings.TrimRight(categoryCrumb, "]")
				category = currentCategory.Title[strings.LastIndex(currentCategory.Title, " ")+1:]
				category = strings.Replace(strings.TrimRight(category, "]"), "-", " ", 1)

				//topNav.WriteString("<li><a href=\"" + siteConfig.BaseURL + "/" + currentSection.Crumb + "/" + categoryCrumb + "/index.html\">" + strings.Replace(categoryCrumb, "-", " ", 1) + "</a>\n<ul>\n")
				topNav.WriteString("<li><a href=\"" + siteConfig.BaseURL + "/" + currentSection.Crumb + "/" + categoryCrumb + "\">" + strings.Replace(categoryCrumb, "-", " ", 1) + "</a>\n<ul>\n")
				//				sectionPageHtml.WriteString("<li><b><a href=\"" + siteConfig.BaseURL + "/" + currentSection.Crumb + "/" + categoryCrumb + "/index.html\">" + strings.Replace(category, "-", " ", 1) + "</a></b></li>\n")
				sectionPageHtml.WriteString("<li><b><a href=\"" + siteConfig.BaseURL + "/" + currentSection.Crumb + "/" + categoryCrumb + "\">" + strings.Replace(category, "-", " ", 1) + "</a></b></li>\n")
				categoryPageHtml.WriteString("<ul>\n")
				sectionPageHtml.WriteString("<ul>\n")
				for _, currentPage := range pages {
					if currentPage.Category == currentCategory.Title {
						categoryUrl := strings.Replace(siteConfig.BaseURL+"/"+currentSection.Crumb+"/"+categoryCrumb+"/"+currentPage.Path[strings.LastIndex(currentPage.Path, "/")+1:], ".html", "", 1)
						fmt.Println(strings.Replace(categoryUrl, ".html", "", 1))
						categoryPageHtml.WriteString("<li><a href=\"" + categoryUrl + "\">" + currentPage.Title + "</a></li>\n")
						sectionPageHtml.WriteString("<li><a href=\"" + categoryUrl + "\">" + currentPage.Title + "</a></li>\n")
						topNav.WriteString("<li><a href=\"" + categoryUrl + "\">" + currentPage.Title + "</a></li>\n")
					}

				}

				categoryPageHtml.WriteString("</ul>\n")
				sectionPageHtml.WriteString("</ul>\n")
				categoryNav := "<a href=\"" + siteConfig.BaseURL + "\">Home</a> // <a href=\"" + siteConfig.BaseURL + "/" + currentSection.Crumb + "\">" + strings.Replace(cases.Title(language.Und).String(currentSection.Crumb), "-", " ", 1) + "</a>" + " // " + strings.Replace(cases.Title(language.Und).String(categoryCrumb), "-", " ", 1)

				//categoryPageUrl := siteConfig.BaseURL + strings.Split(paths.Output, "/out")[1] + "/" + currentSection.Crumb + "/" + categoryCrumb + "/index.html"
				categoryPageUrl := siteConfig.BaseURL + strings.Split(paths.Output, "/out")[1] + "/" + currentSection.Crumb + "/" + categoryCrumb + "/"

				categoryPage := Page{
					Title:       category,
					Content:     template.HTML(categoryPageHtml.String()),
					Path:        paths.Output + "/" + currentSection.Crumb + "/" + categoryCrumb + "/index.html",
					Category:    currentCategory.Title,
					Section:     currentSection.Title,
					Index:       currentSection.Index,
					SiteRoot:    siteConfig.BaseURL,
					Nav:         template.HTML(categoryNav),
					Analytics:   siteConfig.Analytics,
					Description: template.HTML("Notes, ideas, and research I've captured about " + strings.ToLower(category) + "."),
					OgType:      "website",
					Url:         categoryPageUrl,
					OgImage:     siteConfig.BaseURL + "/media/" + siteConfig.OgImage,
				}
				pages = append(pages, categoryPage)
				topNav.WriteString("</ul>\n</li>\n")
			}

		}
		sectionPageHtml.WriteString("</ul>\n")

		sectionNav := "<a href=\"" + siteConfig.BaseURL + "\">Home</a> // " + strings.Replace(cases.Title(language.Und).String(currentSection.Crumb), "-", " ", 1)

		//sectionPageUrl := siteConfig.BaseURL + strings.Split(paths.Output, "/out")[1] + "/" + currentSection.Crumb + "/index.html"
		sectionPageUrl := siteConfig.BaseURL + strings.Split(paths.Output, "/out")[1] + "/" + currentSection.Crumb + "/"

		sectionPage := Page{
			Title:       currentSection.Title,
			Content:     template.HTML(sectionPageHtml.String()),
			Path:        paths.Output + "/" + currentSection.Crumb + "/index.html",
			Category:    currentSection.Title,
			Section:     currentSection.Title,
			Index:       currentSection.Index,
			SiteRoot:    siteConfig.BaseURL,
			Nav:         template.HTML(sectionNav),
			Analytics:   siteConfig.Analytics,
			Description: template.HTML("Notes, ideas, and research I've captured in my " + strings.ToLower(currentSection.Title) + "."),
			OgType:      "website",
			Url:         sectionPageUrl,
			OgImage:     siteConfig.BaseURL + "/media/" + siteConfig.OgImage,
		}
		pages = append(pages, sectionPage)
		topNav.WriteString("</ul></li>")
	}
	return topNav, pages
}

func main() {

	currentDirectory, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	var pages []Page
	var sections []Section
	var categories []Category
	//var siteConfig Config

	siteConfig := siteMeta("./content/.config/config.yaml")

	paths := paths(currentDirectory, siteConfig)

	fmt.Printf("Working Directory: %s\n", currentDirectory)
	fmt.Printf("Content Directory: %s\n", paths.Content)
	fmt.Printf("Output Directory: %s\n", paths.Output)
	fmt.Printf("Template Directory: %s\n", paths.Template)
	fmt.Printf("Asset Directory: %s\n", paths.Asset)

	// To be safe, delete all the output directories and content
	deleteErr := os.RemoveAll(paths.Output)
	if err != nil {
		log.Fatal(deleteErr)
	}

	// Copy over web assets
	filepath.WalkDir(paths.Asset, func(currentFile string, info os.DirEntry, err error) error {
		assetPath := strings.Replace(currentFile, paths.TemplateRoot, "/out", 1)

		if info.IsDir() {
			createDirectory(assetPath)
		} else {
			copyFile(currentFile, assetPath)
		}
		return nil
	})

	filepath.WalkDir(paths.Content, func(currentFile string, info os.DirEntry, err error) error {
		if err != nil {
			log.Fatalf("FATAL ERROR: %s", err.Error())
		}

		// Skip if it's the .config directory or the site.yaml
		// We'll still end up with a .git directory due to subdirectories existing but they'll all be empty - this should be fixed
		if info.Name() == ".config" || info.Name() == "config.yaml" || info.Name() == ".github" || info.Name() == ".git" || info.Name() == "workflows" || info.Name() == "build-site.yaml" {
			log.Print("INFO: Skipping file/path due to rule: `", info.Name())
			return nil
		}

		//skip if it's a .config directory, a .yaml file, or a .git file/directory
		if strings.Contains(currentFile, ".yaml") || strings.Contains(currentFile, ".config") || strings.Contains(currentFile, "README.md") || strings.Contains(currentFile, ".sample") {
			log.Print("INFO: Skipping file/path due to rule: `", info.Name())
			return nil
		}

		outPath := strings.Replace(currentFile, "content", "out", 1)
		outPath = strings.Replace(outPath, ".md", ".html", 1)
		sectionMeta := pageSection(outPath + "/")

		outPath = strings.Replace(outPath, strconv.Itoa(sectionMeta.Index)+"_", "", 1)
		outPath = strings.Replace(outPath, "_", "", 1)

		if info.IsDir() {

			createDirectory(outPath)

		}

		if filepath.Ext(currentFile) == ".md" {
			pages = append(pages, parsePage(currentFile, paths, siteConfig))

			if !slices.Contains(sections, pageSection(currentFile)) {
				sections = append(sections, pageSection(currentFile))
			}

			if !slices.Contains(categories, pageCategory(currentFile)) {
				categories = append(categories, pageCategory(currentFile))
			}

		} else if filepath.Ext(currentFile) != "" {
			copyFile(currentFile, outPath)
		}
		return nil

	})

	topNav, pages := buildNavigation(sections, categories, pages, paths, siteConfig)

	for _, currentPage := range pages {
		createPage(currentPage, sections, template.HTML(topNav.String()), siteConfig)

	}
}
