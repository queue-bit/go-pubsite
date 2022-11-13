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

type Redirects struct {
	Redirect []struct {
		From string `yaml:"from"`
		To   string `yaml:"to"`
	} `yaml:"redirect"`
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
	ChangeFreq  string
	Priority    string
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
	Content          string
	Output           string
	Template         string
	Asset            string
	TemplateSub      string
	TemplateRoot     string
	CurrentDirectory string
}

type TopNav struct {
	Nav template.HTML
}

//Global Vars
var configFile string = "./content/.config/config.yaml"
var redirectFile string = "./content/.config/redirects.yaml"
var siteMeta Config
var siteRedirects Redirects
var sitePaths Paths

//Load config and redirect files
func loadSiteMeta() {

	//--- Get metadata from config file (must exist)
	config := Config{}

	cFile, err := os.Open(configFile)
	if err != nil {
		panic("Could not open configuration file")
	} else {
		log.Println("Loading configuration file.")
	}
	defer cFile.Close()

	// Init and start new YAML decode
	c := yaml.NewDecoder(cFile)

	if err := c.Decode(&config); err != nil {
		siteMeta = Config{}
	}
	siteMeta = config

	//--- Get redirects from file if it exists
	if _, err := os.Stat(redirectFile); err == nil {

		log.Println("Loading redirects file.")

		redirects := Redirects{}

		rfile, err := os.Open(redirectFile)
		if err != nil {
			panic("Redirects file exists but could not open it")
		}
		defer rfile.Close()

		// Init and start new YAML decode
		r := yaml.NewDecoder(rfile)

		if err := r.Decode(&redirects); err != nil {
			siteRedirects = Redirects{}
		}
		siteRedirects = redirects
	} else {
		log.Println("No Redirects file, skipping.")
		return
	}

}

//Setup some default paths
func setPaths() {

	currentDirectory, lpErr := os.Getwd()
	if lpErr != nil {
		log.Fatal(lpErr)
	}

	currentTemplate := "/templates/" + siteMeta.TemplateName
	contentDirectory := currentDirectory + "/content"
	outputDirectory := strings.Replace(contentDirectory, "content", "out", 1)
	templateDirectory := currentDirectory + currentTemplate + "/base/"
	assetDirectory := currentDirectory + currentTemplate + "/assets/"

	sitePaths = Paths{
		Content:          contentDirectory,
		Output:           outputDirectory,
		Template:         templateDirectory,
		Asset:            assetDirectory,
		TemplateSub:      currentTemplate + "/base/",
		TemplateRoot:     currentTemplate,
		CurrentDirectory: currentDirectory,
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

func parsePage(workingFile string) Page {

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
	frontMatter := meta.Get(context)
	pageCategory := pageCategory(workingFile)
	pageSection := pageSection(workingFile)

	var title string
	if frontMatter["title"] == nil {
		title = filepath.Base(outFile)
	} else {
		title = frontMatter["title"].(string)
	}

	var pageIntro string
	if frontMatter["intro"] == nil {
		pageIntro = ""
	} else {
		pageIntro = frontMatter["intro"].(string)
	}

	var pageDescription string
	if frontMatter["description"] == nil {
		pageDescription = ""
	} else {
		pageDescription = frontMatter["description"].(string)
	}

	var ogImage string
	if frontMatter["ogimage"] == nil {
		ogImage = siteMeta.BaseURL + "/media/" + siteMeta.OgImage
	} else {
		ogImage = siteMeta.BaseURL + "/media/" + frontMatter["ogimage"].(string)
	}

	var ogType string
	if frontMatter["ogtype"] == nil {
		ogType = siteMeta.DefaultOgType
	} else {
		ogType = frontMatter["ogtype"].(string)
	}

	var tags string
	if frontMatter["tags"] == nil {
		tags = ""
	} else {
		tags = frontMatter["tags"].(string)
	}

	var articleDate string
	if frontMatter["date"] == nil {
		articleDate = ""
	} else {
		articleDate = frontMatter["date"].(string)
	}

	outFile = strings.Replace(outFile, strconv.Itoa(pageSection.Index)+"_", "", 1)
	outFile = strings.Replace(outFile, "_", "", 1)

	var categoryCrumb string
	categoryCrumb = pageCategory.Crumb[strings.LastIndex(pageCategory.Crumb, " ")+1:]
	categoryCrumb = strings.TrimRight(categoryCrumb, "]")

	pageUrl := siteMeta.BaseURL + strings.Split(outFile, "/out")[1]

	var pageNav string
	if pageSection.Crumb == "" {
		pageNav = ""
	} else {
		pageNav = "<a href=\"" + siteMeta.BaseURL + "\">Home</a> // <a href=\"" + siteMeta.BaseURL + "/" + pageSection.Crumb + "\">"
		pageNav += strings.Replace(cases.Title(language.Und).String(pageSection.Crumb), "-", " ", 1) + "</a>"
		pageNav += " // " + "<a href=\"" + siteMeta.BaseURL + "/" + pageSection.Crumb + "/" + categoryCrumb + "\">"
		pageNav += strings.Replace(cases.Title(language.Und).String(categoryCrumb), "-", " ", 1) + "</a> // " + title
	}

	var canonUrl string

	if pageUrl == siteMeta.BaseURL+"/index.html" {
		canonUrl = siteMeta.BaseURL
	} else {
		canonUrl = strings.Replace(pageUrl, ".html", "", 1)
	}

	return Page{
		Title:       title,
		Content:     template.HTML(buf.String()),
		Path:        outFile,
		Category:    pageCategory.Title,
		Section:     pageSection.Title,
		Index:       pageSection.Index,
		SiteRoot:    siteMeta.BaseURL,
		Nav:         template.HTML(pageNav),
		Intro:       template.HTML(pageIntro),
		Description: template.HTML(pageDescription),
		Analytics:   siteMeta.Analytics,
		Author:      siteMeta.Author,
		OgType:      ogType,
		Url:         canonUrl,
		Date:        articleDate,
		OgImage:     ogImage,
		Tags:        tags,
		ChangeFreq:  "monthly",
		Priority:    "0.5",
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

func createPage(currentPage Page, sections []Section, topNav template.HTML) {
	var templateFiles = []string{"header.html", "footer.html", "body.html", "base.html"}
	var allPaths []string

	file, err := os.Create(currentPage.Path)
	if err != nil {
		log.Fatal("FATAL: ", err, " Could not create ", currentPage.Path, " in createPage(Page,Section,template.HTML,Config)")
	}

	defer file.Close()

	for _, tmpl := range templateFiles {
		allPaths = append(allPaths, "./"+sitePaths.TemplateSub+tmpl)
	}

	templates := template.Must(template.New("").ParseFiles(allPaths...))

	Toc := addToc(string(currentPage.Content), string(currentPage.Title))

	var processed bytes.Buffer
	err = templates.ExecuteTemplate(&processed, "Base", struct{ CurrentPage, SiteMetaData, TopNav, Toc, Sections interface{} }{currentPage, siteMeta, topNav, Toc, sections})

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

				lastLevel, tocLineItem = tocLevels(level, lastLevel, tocLinkItem)

				if tocLineItem != "" {
					lastToc = lastToc + tocLineItem
					count++
				}
			}
		}
	}

	var closeTags string

	// We need to close out the lists (ul's and li's) that were opened
	for i := lastLevel; i > 0; i-- {
		closeTags = closeTags + "<!--cbs--></li></ul>"
	}

	if count >= 3 {
		return template.HTML(lastToc + closeTags + "<br/>\n\n")
	} else {
		return template.HTML("")
	}

}

func tocLevels(level int, lastLevel int, tocLinkItem string) (int, string) {
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

func buildNavigation(sections []Section, categories []Category, pages []Page) (strings.Builder, []Page) {
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
		topNav.WriteString("<li>\n<a href=\"" + siteMeta.BaseURL + "/" + currentSection.Crumb + "\">" + strings.Replace(currentSection.Crumb, "-", " ", 1) + "</a>\n<ul>\n")
		for _, currentCategory := range categories {
			categoryPageHtml.Reset()
			parentCategory = currentCategory.Parent[strings.LastIndex(currentCategory.Parent, " ")+1:]
			parentCategory = strings.TrimRight(parentCategory, "]")

			if parentCategory == currentSection.Crumb {
				categoryCrumb = currentCategory.Crumb[strings.LastIndex(currentCategory.Crumb, " ")+1:]
				categoryCrumb = strings.TrimRight(categoryCrumb, "]")
				category = currentCategory.Title[strings.LastIndex(currentCategory.Title, " ")+1:]
				category = strings.Replace(strings.TrimRight(category, "]"), "-", " ", 1)

				topNav.WriteString("<li><a href=\"" + siteMeta.BaseURL + "/" + currentSection.Crumb + "/" + categoryCrumb + "\">" + strings.Replace(categoryCrumb, "-", " ", 1) + "</a>\n<ul>\n")
				sectionPageHtml.WriteString("<li><b><a href=\"" + siteMeta.BaseURL + "/" + currentSection.Crumb + "/" + categoryCrumb + "\">" + strings.Replace(category, "-", " ", 1) + "</a></b></li>\n")
				categoryPageHtml.WriteString("<ul>\n")
				sectionPageHtml.WriteString("<ul>\n")
				for _, currentPage := range pages {
					if currentPage.Category == currentCategory.Title {
						categoryUrl := strings.Replace(siteMeta.BaseURL+"/"+currentSection.Crumb+"/"+categoryCrumb+"/"+currentPage.Path[strings.LastIndex(currentPage.Path, "/")+1:], ".html", "", 1)
						categoryPageHtml.WriteString("<li><a href=\"" + categoryUrl + "\">" + currentPage.Title + "</a></li>\n")
						sectionPageHtml.WriteString("<li><a href=\"" + categoryUrl + "\">" + currentPage.Title + "</a></li>\n")
						topNav.WriteString("<li><a href=\"" + categoryUrl + "\">" + currentPage.Title + "</a></li>\n")
					}

				}

				categoryPageHtml.WriteString("</ul>\n")
				sectionPageHtml.WriteString("</ul>\n")
				categoryNav := "<a href=\"" + siteMeta.BaseURL + "\">Home</a> // <a href=\"" + siteMeta.BaseURL + "/" + currentSection.Crumb + "\">" + strings.Replace(cases.Title(language.Und).String(currentSection.Crumb), "-", " ", 1) + "</a>" + " // " + strings.Replace(cases.Title(language.Und).String(categoryCrumb), "-", " ", 1)
				categoryPageUrl := siteMeta.BaseURL + strings.Split(sitePaths.Output, "/out")[1] + "/" + currentSection.Crumb + "/" + categoryCrumb + "/"

				categoryPage := Page{
					Title:       category,
					Content:     template.HTML(categoryPageHtml.String()),
					Path:        sitePaths.Output + "/" + currentSection.Crumb + "/" + categoryCrumb + "/index.html",
					Category:    currentCategory.Title,
					Section:     currentSection.Title,
					Index:       currentSection.Index,
					SiteRoot:    siteMeta.BaseURL,
					Nav:         template.HTML(categoryNav),
					Analytics:   siteMeta.Analytics,
					Description: template.HTML("Notes, ideas, and research I've captured about " + strings.ToLower(category) + "."),
					OgType:      "website",
					Url:         categoryPageUrl,
					OgImage:     siteMeta.BaseURL + "/media/" + siteMeta.OgImage,
					ChangeFreq:  "weekly",
					Priority:    "0.8",
				}
				pages = append(pages, categoryPage)
				topNav.WriteString("</ul>\n</li>\n")
			}

		}
		sectionPageHtml.WriteString("</ul>\n")

		sectionNav := "<a href=\"" + siteMeta.BaseURL + "\">Home</a> // " + strings.Replace(cases.Title(language.Und).String(currentSection.Crumb), "-", " ", 1)
		sectionPageUrl := siteMeta.BaseURL + strings.Split(sitePaths.Output, "/out")[1] + "/" + currentSection.Crumb + "/"

		sectionPage := Page{
			Title:       currentSection.Title,
			Content:     template.HTML(sectionPageHtml.String()),
			Path:        sitePaths.Output + "/" + currentSection.Crumb + "/index.html",
			Category:    currentSection.Title,
			Section:     currentSection.Title,
			Index:       currentSection.Index,
			SiteRoot:    siteMeta.BaseURL,
			Nav:         template.HTML(sectionNav),
			Analytics:   siteMeta.Analytics,
			Description: template.HTML("Notes, ideas, and research I've captured in my " + strings.ToLower(currentSection.Title) + "."),
			OgType:      "website",
			Url:         sectionPageUrl,
			OgImage:     siteMeta.BaseURL + "/media/" + siteMeta.OgImage,
			ChangeFreq:  "weekly",
			Priority:    "1",
		}
		pages = append(pages, sectionPage)
		topNav.WriteString("</ul></li>")
	}
	return topNav, pages
}

//Return a single sitemap item (for one url)
func sitemap(currentPage Page) string {
	siteMapItem := "  <url>\n"
	siteMapItem += "    <loc>" + currentPage.Url + "</loc>\n"
	siteMapItem += "    <changefreq>" + currentPage.ChangeFreq + "</changefreq>\n"
	siteMapItem += "    <priority>" + currentPage.Priority + "</priority>\n"
	siteMapItem += "  </url>\n"
	return siteMapItem
}

func createRedirects() {

	log.Println("Creating Redirect Files")
	for _, currentRedirect := range siteRedirects.Redirect {
		toUrl := siteMeta.BaseURL + currentRedirect.To
		filePath := sitePaths.Output + currentRedirect.From

		createDirectory(sitePaths.Output + "/" + currentRedirect.From)

		html := "<html><head><meta http-equiv=\"refresh\" content=\"0;URL=" + toUrl + "\"></head><body><h1>Page has moved</h1><p>If not automatically redirected <a href=\"" + toUrl + "\">please click here</a>.</p></body></html>"

		file, err := os.Create(filePath + "index.html")
		//log.Println("\t", filePath+"index.html")
		if err != nil {
			log.Fatal("FATAL: ", err, " Could not create ", filePath+".html", " in createRedirects()")
		}

		w := bufio.NewWriter(file)
		w.WriteString(html)
		w.Flush()
		defer file.Close()
	}
}

func main() {

	loadSiteMeta()
	setPaths()

	var pages []Page
	var sections []Section
	var categories []Category

	log.Println("Working Directory:\t", sitePaths.CurrentDirectory)
	log.Println("Content Directory:\t", sitePaths.Content)
	log.Println("Output Directory:\t", sitePaths.Output)
	log.Println("Template Directory:\t", sitePaths.Template)
	log.Println("Asset Directory:\t", sitePaths.Asset)

	// To be safe, delete all the output directories and content
	deleteErr := os.RemoveAll(sitePaths.Output)
	if deleteErr != nil {
		log.Fatalf("FATAL ERROR: %s", deleteErr)
	}

	// Copy over web assets
	log.Println("Copying web assets")
	filepath.WalkDir(sitePaths.Asset, func(currentFile string, info os.DirEntry, walkErr error) error {
		if walkErr != nil {
			log.Fatalf("FATAL ERROR: %s", walkErr.Error())
		}
		assetPath := strings.Replace(currentFile, sitePaths.TemplateRoot, "/out", 1)

		//log.Println("\t", assetPath)
		if info.IsDir() {
			createDirectory(assetPath)
		} else {
			copyFile(currentFile, assetPath)
		}
		return nil

	})

	filepath.WalkDir(sitePaths.Content, func(currentFile string, info os.DirEntry, err error) error {
		if err != nil {
			log.Fatalf("FATAL ERROR: %s", err.Error())
		}

		// Skip if it's the .config directory or the site.yaml
		// We'll still end up with a .git directory due to subdirectories existing but they'll all be empty - this should be fixed
		if info.Name() == ".config" || info.Name() == "config.yaml" || info.Name() == ".github" || info.Name() == ".git" || info.Name() == "workflows" || info.Name() == "build-site.yaml" {
			//log.Print("INFO: Skipping file/path due to rule: `", info.Name())
			return nil
		}

		//skip if it's a .config directory, a .yaml file, or a .git file/directory
		if strings.Contains(currentFile, ".yaml") || strings.Contains(currentFile, ".config") || strings.Contains(currentFile, "README.md") || strings.Contains(currentFile, ".sample") {
			//log.Print("INFO: Skipping file/path due to rule: `", info.Name())
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
			pages = append(pages, parsePage(currentFile))

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

	topNav, pages := buildNavigation(sections, categories, pages)

	var siteMap string = ""

	for _, currentPage := range pages {
		createPage(currentPage, sections, template.HTML(topNav.String()))
		siteMap += sitemap(currentPage)
	}

	// Create a sitemap.xml file

	siteMapSchema := "<urlset xmlns:xsi=\"https://www.w3.org/2001/XMLSchema-instance\" xsi:schemaLocation=\"https://www.sitemaps.org/schemas/sitemap/0.9 https://www.sitemaps.org/schemas/sitemap/0.9/sitemap.xsd\" xmlns=\"https://www.sitemaps.org/schemas/sitemap/0.9\">\n"
	siteMapSchema += siteMap + "</urlset>"

	file, err := os.Create(sitePaths.Output + "/sitemap.xml")
	if err != nil {
		log.Fatal("FATAL: ", err, " Could not create ", sitePaths.Output+"/sitemap.xml", " in main()")
	}

	w := bufio.NewWriter(file)
	w.WriteString(siteMapSchema)
	w.Flush()
	defer file.Close()

	createRedirects()
}
