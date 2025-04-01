#!/usr/local/bin/venv_ask/bin/python3
# Version 0.1.4
# Check variable names when editing an existing snippet

import sqlite3
from tabulate import tabulate
import json
import argparse
import glob
import sys
import yaml
import os
import re

#TODO - better search, smart parsing.
#static dump folder
#conf file -- database, rawSnippetsDir, UltiSnipsDir

def parse_args():
	parser = argparse.ArgumentParser()
	#subparsers = parser.add_subparsers(title='subcommands', description='sub-command help')
	# Define a parser for the 'add' subcommand
	group = parser.add_mutually_exclusive_group(required=True)
	group.add_argument('--search')
	group.add_argument('--update',action="store_true")
	group.add_argument('--edit')
	group.add_argument('--browse',action="store_true")
	group.add_argument('--remove')
	group.add_argument('--render-all',action="store_true")
	group.add_argument('--prune',action="store_true")
	args = parser.parse_args()
	return(args)

def check_valid_snippet(snippetName):
    cur.execute(f"SELECT * FROM snippets WHERE name = '{snippetName}'")
    try:
        cur.fetchall()[0][0]
    except:
        print(f'[!] {snippetName} not found in database.')
        sys.exit()
    return

def fetchTags(id,cur):
	cur.execute(f"SELECT tagId FROM tagMap WHERE snipId= {id};")
	tagIds = cur.fetchall()
	tags = []
	for tagId in tagIds:
		cur.execute(f"SELECT tag FROM tags WHERE id = '{tagId[0]}'")
		tag = cur.fetchall()[0][0]
		tags.append(tag)
	tags = ",".join(tags)
	return(tags)

def vim_edit_files(yml,snippet):
    editor = os.environ.get('EDITOR', 'vim')
    input(f"[!] Now you can edit the snippet metadata for {snippet} using {editor}.")
    os.system(f"{editor} {yml} {snippet}")
    return

def edit(snippetId,cur):
    #See if we have a snippet id or name
    try:
        int(snippetId)
    except ValueError:
        check_valid_snippet(snippetId)
        snippetId = getId(snippetId,cur)
    #Get snippet data
    cur.execute(f"SELECT * FROM snippets WHERE id = '{snippetId}'")
    try:
        _id,snippetName,description,variableString,version,snippetText = cur.fetchall()[0]
        tags = fetchTags(snippetId,cur)
    except:
        print(f"[-] Failed to find '{snippetName}' in the database.")
        sys.exit()
    tempYamlFile = f"{snippetName}.yaml"
    updated_version = round(float(version) + float(0.1),2)
    write_yaml(snippetName,description,variableString,tags,updated_version)
    snippetFile=write_snippetFile(snippetName,snippetText)
    vim_edit_files(tempYamlFile,snippetFile)
    if update_db(tempYamlFile,cur,snippetFile):
        # Clean up the text file
        os.remove(tempYamlFile)
        os.remove(snippetFile)
    else:
        print(f"[!] Failed to update {snippetName}")
    return()



def connect():
	if os.path.exists(db_path):
		print("[+] Database exists")
	else:
		print("[!] Database does not exist, creating it")
		create_db()
	conn = sqlite3.connect(db_path)
	cur = conn.cursor()
	return(conn,cur)

def create_db():
	# Create a table with 6 fields
	conn = sqlite3.connect(db_path)
	cur = conn.cursor()
	cur.execute("""
		CREATE TABLE IF NOT EXISTS snippets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			description TEXT,
			variables TEXT,
			version TEXT,
			snippetText TEXT
		)""")
	# Commit the changes
	conn.commit()
	cur.execute("""
		CREATE TABLE IF NOT EXISTS tagMap (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tagId INTEGER,
			snipId INTEGER,
		)""")
	# Commit the changes
	conn.commit()

	cur.execute("""
		CREATE TABLE IF NOT EXISTS tags (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tag TEXT,
		)""")
	# Commit the changes
	conn.commit()
	conn.close()
	print("Created database")

	return()

def read_yaml(yamlFile):
    with open(yamlFile,"r") as f:
        yamlData = yaml.safe_load(f)
    return(yamlData)

def varNameStandard(variableDict):
    variables = list(set(variableDict.keys()))
    fixedDict = {}
    translatedVars = {}
    for variable in variables:
	    fixedName = variable.upper().replace(" ","_")
	    fixedDict[fixedName] = variableDict[variable]
	    translatedVars[variable] = fixedName
    return(fixedDict,translatedVars)

def parse_yaml(yamlDict):
    name = yamlDict['name']
    description = yamlDict['description']
    variableDict = yamlDict['variables']
# fix var names
    variableDict,translatedVars = varNameStandard(variableDict)
    try:
	    version = yamlDict['version']
    except:
	    version = "0.1"
    try:
	    tags = yamlDict['tags']
    except:
	    tags = "Untagged"
    return(name, description, variableDict, version, translatedVars,tags)



def parseSnippet(variableDict,translatedVars,snippetFile,yamlFile):
    #open file
    with open(snippetFile,"r") as f:
        snippetText = f.read()
    #change out old var names
    for oldName in translatedVars.keys():
        newName = translatedVars[oldName]
        snippetText = snippetText.replace(oldName, newName)
    #lint file
    #snippetText = snippetText.replace('$', '\\$')
    #look for yaml vars in text
    variableList = list(dict(variableDict).keys())
    foundVar=[]
    for snipVar in variableList:
        if snipVar in snippetText:
            foundVar.append(snipVar)
    for varFound in foundVar:
        variableList.remove(varFound)
    if len(variableList) != 0:
        print(f"[!] {snippetFile} does not use the following variables: {','.join(variableList)}")
        vim_edit_files(yamlFile,snippetFile)
        parseSnippet(variableDict,translatedVars,snippetFile,yamlFile)
    #look for text vars in yaml
    pattern = r'\{ ([a-zA-Z0-9_]+) \}'
    matches = re.findall(pattern, snippetText)
    foundMatch = []
    missingVar = []
    for match in matches:
        if match in foundVar:
            foundMatch.append(match)
        else:
            missingVar.append(match)
    if len(missingVar) != 0:
        print(f"[!] {snippetFile}.yaml does not use the following variable(s): {','.join(missingVar)}")
        vim_edit_files(yamlFile,snippetFile)
        yamlDict=read_yaml(yamlFile)
        # parse yaml
        name,description, variableDict, version, translatedVars,tags = parse_yaml(yamlDict)
        parseSnippet(variableDict,translatedVars,snippetFile,yamlFile)
    print("[+] Variable numbers match -- linting complete")
    return(snippetText)

def update_db(yaml,cur,snippetFile):
    # read yaml
    yamlDict=read_yaml(yaml)
    # parse yaml
    name,description, variableDict, version, translatedVars,tags = parse_yaml(yamlDict)
    # parse snippet
    snippetText = parseSnippet(variableDict, translatedVars,snippetFile,yaml)
    # update db
    ##check version
    cur.execute(f"SELECT * FROM snippets WHERE name LIKE '%{name}%' ORDER BY version DESC")
    try:
        _id,_snippetName,_description,_variableString,oldVersion,_snippetText = cur.fetchall()[0]
    except:
        oldVersion = "0"
    if float(version) > float(oldVersion):
        sqlString = f"INSERT INTO snippets (name, description, variables, version, snippetText) VALUES (?,?,?,?,?)"
        cur.execute(sqlString,(name,description,json.dumps(variableDict),version, snippetText))
        conn.commit()
        print(f"[+] Wrote {name}_v{version} to database")
        updateSuccess = True
    else:
        print(f"[-] Failed to update. Database version of {name} is {oldVersion}.  Update version number in {yaml} to greater than {oldVersion} to update")
        updateSuccess = False
    if updateSuccess == True and tags != "Untagged":
        for tag in tags.split(","):
            updateTags(conn,cur,tag,name)
    return(updateSuccess)

def updateTags(conn,cur,tag,snippetName):
	#check if tag exists
	cursor = conn.execute("SELECT id FROM tags WHERE tag = ?", (tag,))
	# Check if the result is not empty
	try:
		cursor.fetchall()
		tagId=int(cursor.fetchall()[0][0])
	except:
		conn.execute("INSERT INTO tags (tag) VALUES (?)",(tag,))
		conn.commit()
		cursor = conn.execute("SELECT id FROM tags WHERE tag = ?", (tag,))
		tagId=int(cursor.fetchall()[0][0])
	#link tag to snippet
	cursor = conn.execute("SELECT id FROM snippets WHERE name = ? ORDER BY version DESC", (snippetName,))
	snipId=int(cursor.fetchall()[0][0])
	conn.execute("INSERT INTO tagMap (tagId, snipId) VALUES (?,?)",(tagId,snipId,))
	conn.commit()
	print(f"[+] Mapped {tag} to {snippetName}.")
	return()

def getId(snippetName,cur):
    try:
        if isinstance(int(snippetName),int):
            snipId = int(snippetName)
    except ValueError:
        #Need to see if the snippet even exists
        cur.execute(f"SELECT id FROM snippets WHERE name LIKE '%{snippetName}%' ORDER BY version DESC")
        snipId = cur.fetchall()[0][0]
    return(snipId)

def getVersion(snippetName,cur):
    cur.execute(f"SELECT version FROM snippets WHERE name = '{snippetName}' ORDER BY version DESC")
    snipVersion = cur.fetchall()[0][0]
    return(snipVersion)


def dump(snipId):
    cur.execute(f"SELECT * FROM snippets WHERE id = '{snipId}' ORDER BY version DESC")
    _id,snippetName,description,variableString,version,snippetText = cur.fetchall()[0]
    tags = fetchTags(_id,cur)
    snippetFile=write_snippetFile(snippetName,snippetText)
    write_yaml(snippetName,description,variableString,tags,version)
    print(f"[+] Wrote {snippetFile} and {snippetName}.yaml")
    print(f"[+] When you are ready to update the database, run \n ./ask.py --update {snippetName}.yaml")
    return

def write_snippetFile(snippetName,snippetText):
	snippetFile=f"{snippetName}.txt"
	with open(snippetFile,"w") as f:
		f.write(snippetText)
	return (snippetFile)

def write_yaml(snippetName, description, variableString, tags,version):
    yamlDict = {"name":snippetName,"version":version,"description":description, "variables":json.loads(variableString),"tags":tags}
    with open(f"{snippetName}.yaml","w") as f:
        yaml.dump(yamlDict, f)
    return ()

def render(snippetName):
    #lookup snippet info
    snipId = getId(snippetName,cur)
    cur.execute(f"SELECT * FROM snippets WHERE id = '{snipId}'")#e ORDER BY version DESC")
    try:
        _id,snippetName,description,variableString,version,snippetText = cur.fetchall()[0]
    except:
        print(f"[-] Failed to find '{snippetName}' in the database.")
        sys.exit()
    #prepare variables
    variableDict = json.loads(variableString)
    #prepare snippetText
    snippetText = snippetText.replace('`','\\`')
    #put it together
    header = f"snippet {snippetName} \"{description}\"\n"
    table_data = [\
        ["FIELD", "VALUE"],
        ["Snippet Name ", snippetName],
        ["Version ",version],
        ["Description  ", description],
        ]
    table = tabulate(table_data,headers="firstrow",tablefmt="github")
    metaSection=table+"\n"*2
    if variableDict != {}:
        indexValue=1
        tabDict={}
        table_data = [\
            ["VARIABLE", "TAB STOP","EXAMPLE VALUE","DESCRIPTION"],
            ]
    for variable in variableDict.keys():
        if "example" not in variableDict[variable]:
            variableDict[variable]['example']='no example'
        if "description" not in variableDict[variable]:
            variableDict[variable]['description']='no description'
        if variableDict[variable].get('default') is None:
            table_data.append([variable,f"${{{str(indexValue*5)}:{variable}}}",variableDict[variable]['example'],variableDict[variable]['description']])
        elif '|' in variableDict[variable]['default']:
            choices = ','.join(variableDict[variable]['default'].split('|'))
            table_data.append([variable,f"${{{str(indexValue*5)}|{choices}|}}",variableDict[variable]['example'],variableDict[variable]['description']])
        elif "example" in variableDict[variable].keys():
            table_data.append([variable,f"${{{str(indexValue*5)}:{variableDict[variable]['default']}}}",variableDict[variable]['example'],variableDict[variable]['description']])
        else:
            table_data.append([variable,f"${{{str(indexValue*5)}:{variableDict[variable]['default']}}}","No example given",variableDict[variable]['description']])
        snippetText = snippetText.replace(f"{{ {variable} }}","$"+str(indexValue*5))
        indexValue+=1
    table = tabulate(table_data,headers="firstrow",tablefmt="github",maxcolwidths=[None, None, None, 30])
    metaSection+=table+"\n"*2
    snippetSection = "\t" + "#" * 12 + " Snippet " + "#" * 17 + "\n$0"
    snippetSection += snippetText
    fullSnip = header + metaSection + snippetSection +"endsnippet\n"
    with open(f"{UltiSnipsDir}{snippetName}_v{version}.snippets","w") as f:
        f.write(fullSnip)
    print(f"[+] Rendered {snippetName}_v{version}.snippets")
    return()

def process_newSnippet(snippet,version,inputFile):
    with open(inputFile,"r") as f:
        rawText = f.read()
    #Parse out variables
    pattern = r'\{ ([a-zA-Z0-9_]+) \}'
    matches = re.findall(pattern, rawText)
    variables = list(set(matches))
    #write yaml
    snippetName = input(f"What is the name of the snippet?  Maybe something like [{snippet}]. ")
    if snippetName == "":
        snippetName = snippet
    elif " " in snippetName:
        snippetName = snippetName.replace(' ','_')
        print(f'[+] Snippet cannot have spaces in the name.  Converting to {snippetName}.')
    description = input("What does the snippet do? ")
    if description == "":
        description = "No description given."
    tags = input("Any tags you want associated? Use coma delimmiter. ")
    variableDict = {}
    for variable in variables:
        varDesc=input(f"What is the {variable} used for and how do you get it? ")
        varExample=input(f"What is an example value of {variable}? ")
        #Could put in some logic here to look for options instead of a default?
        varDefault=input(f"What is an default value of {variable}? For multiple choices, provide | between choices.")
        variableDict[variable]={"description":varDesc,"example":varExample,"default":varDefault}
    yamlDict={"name":snippetName,"description":description,"snippetFile":inputFile,"variables":variableDict,"tags":tags,'version':version}
    yamlFile = f"{snippetName}_v{version}.yaml"
    with open(yamlFile,"w") as f:
        yaml.dump(yamlDict,f)
    update_db(yamlFile,cur,inputFile)
    os.remove(yamlFile)
    print(f"[+] Added new snippet: {snippetName}~v{version}")
    return(snippetName)


def update(cur):
    #look in snippets folder
    pattern = f"{rawSnippetsDir}*.txt"
    files = glob.glob(pattern)
    #determine if we have this snippet and version
    snipList=[]
    for snippetFile in files:
        #Check if the raw snippet files are named appropriately, and create a list
        #This should be better.  should be formatted as {snippetname}~v{version}.txt
        try:
            basename = os.path.basename(snippetFile)
            snippetName = basename.split('~')[0]
            snipVersion = basename.split('~')[1][:-4][1:]
            snipList.append(f"{snippetName}~{snipVersion}")
        #If it isn't named appropriately, it is probably new.
        except Exception as e:
            print(f"[!] New snippet detected - {snippetFile} .  Does not appear to be named correctly.")
            print(e)
            version = "0.10"
            # Read snippet file
            inputFile = f"{rawSnippetsDir}/{snippetName}~v{version}.txt"
            snippetName = process_newSnippet(snippetFile,version,inputFile)
            #Rename snippet file appropriately.
            snipList.append(f"{snippetName}~0.10")
            os.system(f"mv {rawSnippetsDir}/{snippetFile} {rawSnippetsDir}/{snippetName}~v0.10.txt")
    #Look at all the existing snippets
    cur.execute(f"SELECT name,version FROM snippets ")
    results = cur.fetchall()
    db_snipList= []
    for result in results:
        snippetName , snipVersion = result
        db_snipList.append(f"{snippetName}~{snipVersion}")
    #Compare what our raw snippets with our existing snippets to find what is new.
    newSnips = [ snippet for snippet in snipList if snippet not in db_snipList ]
    for snippet in newSnips:
        snippetName,version = snippet.split('~')
        try:
            latestVersion= float(getVersion(snippetName,cur))
            if float(version) > latestVersion:
                cur.execute(f"SELECT * FROM snippets WHERE name = '{snippetName}' ORDER BY version DESC")
                id,snippetName,description,variableString,oldVersion,snippetText = cur.fetchall()[0]
                tags=fetchTags(id,cur)
                snippetFile = f"{rawSnippetsDir}{snippetName}~v{version}.txt"
                write_yaml(snippetName, description, variableString, tags, version)
                tempYamlFile = f"{snippetName}.yaml"
                # need to check if the variables match up in the database and the new file
                update_db(tempYamlFile,cur,snippetFile)
                os.remove(tempYamlFile)
                print(f"[+] Updated old snippet: {snippetName}")
#            else:
#                print(f"[!] Unable to update {snippetName}~v{version}.  Existing version is {latestVersion}.")
        except Exception as e:
            print(e)
            print(f"[!] New snippet detected: {snippetName}")
            inputFile = f"{rawSnippetsDir}/{snippetName}~v{version}.txt"
            process_newSnippet(snippetName,version,inputFile)
    return()

def search_db(queryTerm,cur):
	#search
	cur.execute(f"SELECT * FROM snippets WHERE name LIKE '%{queryTerm}%'")
	results = cur.fetchall()
	#make it pretty
	table_data=[]
	for result in results:
		id,snippetName,description,variableString,version,_snippetText = result
		table_data.append([id,snippetName,version,description])
	table = tabulate(table_data,headers="firstrow",tablefmt="simple_grid")
	print(table)
	return()

def remove(id,cur,conn):
    cur.execute(f"DELETE FROM snippets WHERE id = '{id}'")
    conn.commit()
    try:
        cur.execute(f"DELETE tagId FROM tagMap WHERE snipId= {id};")
    except:
        print(f"[+] No associated tags")
    print(f"[+] Removed")
    return()

def browse(cur):
    #Show what we have
    cur.execute(f"SELECT id, name, version, description FROM snippets")
    results = cur.fetchall()
    conn.close()
    table_data = [["ID", "SNIPPET", "VERSION", "DESCRIPTION"]]
    for result in results:
        id, name, version, description = result
        table_data.append([id,name,version,description])
    table = tabulate(table_data,headers="firstrow",tablefmt="github")
    print(table)
    return()

def render_all(cur):
    #remove all the old snippets
    os.system(f"rm -rf {UltiSnipsDir}/*")
    #find the highest version of each snippet
    sqlLine="SELECT id, name, version FROM snippets WHERE (name, version) IN ( SELECT name, MAX(version) FROM snippets GROUP BY name);"
    cur.execute(sqlLine)
    results = cur.fetchall()
    #render it
    for result in results:
        id, name, version = result
        try:
            render(name)
        except Exception as e:
            print(f"Failed trying to render {name}: {e}")
    print("[+] Rendered all snippets")
    return()

def prune(cur):
    #find the highest version of each snippet
    sqlLine="SELECT id FROM snippets WHERE (name, version) IN ( SELECT name, MAX(version) FROM snippets GROUP BY name);"
    cur.execute(sqlLine)
    results = cur.fetchall()
    preserveIds = []
    for snipId in results:
        preserveIds.append(snipId[0])
    sqlLine="SELECT id FROM snippets ;"
    cur.execute(sqlLine)
    results = cur.fetchall()
    allIds = []
    for snipId in results:
        allIds.append(snipId[0])
    for snipId in allIds:
        if snipId not in preserveIds:
            remove(snipId,cur,conn)
    print("[+] Pruned old versions")
    return()



db_path = '/root/gits/ask/mydatabase.db'        #where the database is located
UltiSnipsDir = "/root/gits/Confs/all/"           #where the UltiSnips should go
rawSnippetsDir = '/root/gits/ask/snippets/'        #Where the raw snippets are

if __name__ == '__main__':
    args = parse_args()
    conn,cur= connect()
    if args.search:
        search_db(args.search,cur)
    elif args.update:
        update(cur)
    elif args.edit:
        edit(args.edit,cur)
    elif args.browse:
        browse(cur)
    elif args.remove:
        remove(args.remove,cur,conn)
    elif args.render_all:
        render_all(cur)
    elif args.prune:
        prune(cur)
    conn.close()
    print("[+] Database closed")
